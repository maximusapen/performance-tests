/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdzone

import (
	"fmt"

	"github.com/urfave/cli"

	"github.ibm.com/alchemy-containers/armada-api-model/json/v2/requests"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

// satelliteZoneAdd will add a zone to a worker pool in a satellite cluster
func satelliteZoneAdd(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	suffix := c.Bool(models.SuffixFlagName)

	quantity, err := cliutils.GetFlagInt(c, models.QuantityFlagName, true, "Number of clusters", nil)
	if err != nil {
		return err
	}

	workerPool, err := cliutils.GetFlagString(c, models.WorkerPoolFlagName, false, "Name of the worker pool to which the zone should be added.", nil)
	if err != nil {
		return err
	}

	zone, err := cliutils.GetFlagString(c, models.ZoneFlagName, false, "Name of the zone to be added.", nil)
	if err != nil {
		return err
	}

	zoneRequest := requests.SatelliteWorkerPoolZoneAdd{
		CommonWorkerPoolZoneAdd: requests.CommonWorkerPoolZoneAdd{
			ID:         zone,
			Cluster:    clusterNameOrID,
			WorkerPool: workerPool,
		},
	}

	endpoint := resources.GetArmadaV2Endpoint(c)

	for cl := 1; cl <= quantity; cl++ {
		if suffix || quantity > 1 {
			zoneRequest.Cluster = fmt.Sprintf("%s%d", clusterNameOrID, cl)
		}

		if err = endpoint.AddSatelliteZone(zoneRequest); err != nil {
			return err
		}

		fmt.Fprintf(c.App.Writer, "\nAddition of zone '%s' request for Satellite cluster '%s' successful.\n", zone, zoneRequest.Cluster)
	}

	return nil
}
