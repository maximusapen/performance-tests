/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdworkerpool

import (
	"fmt"
	"strconv"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-api-model/json/v2/requests"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

const (
	satelliteFlavorUPI = "upi"
)

// satelliteWorkerPoolCreate creates a worker pool.
func satelliteWorkerPoolCreate(c *cli.Context) error {
	poolName, err := cliutils.GetFlagString(c, models.NameFlagName, false, "Name of the worker pool to be created.", nil)
	if err != nil {
		return err
	}

	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	zoneID, err := cliutils.GetFlagString(c, models.ZoneFlagName, false, models.ZoneFlagValue, nil)
	if err != nil {
		return err
	}

	suffix := c.Bool(models.SuffixFlagName)

	quantity, err := cliutils.GetFlagInt(c, models.QuantityFlagName, true, "Number of clusters", nil)
	if err != nil {
		return err
	}

	sizePerZoneString, err := cliutils.GetFlagString(c, models.SizePerZoneFlagName, false, "Number of workers per zone", nil)
	if err != nil {
		return err
	}
	sizePerZone, err := strconv.ParseInt(sizePerZoneString, 10, 64)
	if err != nil {
		return err
	}

	labelsMap, err := cliutils.MakeLabelsMap(c)
	if err != nil {
		return err
	}

	hostLabelsMap, err := cliutils.MakeHostLabelsMap(c)
	if err != nil {
		return err
	}

	osName, err := cliutils.GetFlagString(c, models.OperatingSystemFlagName, true, "Worker Pool Operating Systsem", nil)
	if err != nil {
		return err
	}

	// Set the properties of the desired worker pool
	workerPoolProperties := requests.SatelliteCreateWorkerPool{
		CommonCreateWorkerPool: requests.CommonCreateWorkerPool{
			Cluster:         clusterNameOrID,
			Name:            poolName,
			Flavor:          satelliteFlavorUPI,
			WorkerCount:     int(sizePerZone),
			Labels:          labelsMap,
			OperatingSystem: osName,
		},
		SatelliteCreateWorkerPoolFragment: requests.SatelliteCreateWorkerPoolFragment{
			HostLabels: hostLabelsMap,
		},
		Zones: []requests.SatelliteCreateWorkerPoolZone{
			{
				CommonCreateWorkerPoolZone: requests.CommonCreateWorkerPoolZone{
					ID: zoneID,
				},
			},
		}}

	endpoint := resources.GetArmadaV2Endpoint(c)

	for cl := 1; cl <= quantity; cl++ {
		if suffix || quantity > 1 {
			workerPoolProperties.Cluster = fmt.Sprintf("%s%d", clusterNameOrID, cl)
		}

		if err := endpoint.CreateSatelliteWorkerPool(workerPoolProperties); err != nil {
			return err
		}

		fmt.Fprintf(c.App.Writer, "\nWorker Pool creation request for satellite cluster '%s' successful.\n", workerPoolProperties.Cluster)
	}

	return nil
}
