/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2020, 2022 All Rights Reserved.
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

func vpcClassicWorkerPoolCreate(c *cli.Context) error {
	return vpcWorkerPoolCreate(c, models.ProviderVPCClassic, cliutils.GetInfrastructureConfig().VPCClassic)
}
func vpcNextGenWorkerPoolCreate(c *cli.Context) error {
	return vpcWorkerPoolCreate(c, models.ProviderVPCGen2, cliutils.GetInfrastructureConfig().VPC)
}

// vpcWorkerPoolCreate creates a worker pool.
// Note that we're not actually using the provider (gen1 / gen2) as that is used when adding zones to the pool.
// That's currently done by the zone add command but could be introduced here to combine the two
// user operations into one.
func vpcWorkerPoolCreate(c *cli.Context, provider string, infraConfig *cliutils.TomlVPCInfrastructureConfig) error {
	poolName, err := cliutils.GetFlagString(c, models.NameFlagName, false, "Name of the worker pool to be created.", nil)
	if err != nil {
		return err
	}

	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	suffix := c.Bool(models.SuffixFlagName)

	quantity, err := cliutils.GetFlagInt(c, models.QuantityFlagName, true, "Number of clusters", nil)
	if err != nil {
		return err
	}

	vpcID, err := cliutils.GetFlagString(c, models.VPCIDFlagName, false, "Virtual Private Cloud Identifier", infraConfig.ID)
	if err != nil {
		return err
	}

	flavor, err := cliutils.GetFlagString(c, models.MachineTypeFlagName, false, "Worker Node Machine Flavor", infraConfig.Flavor)
	if err != nil {
		return err
	}

	osName, err := cliutils.GetFlagString(c, models.OperatingSystemFlagName, true, "Operating System Name", nil)
	if err != nil {
		return err
	}

	isolationType, err := cliutils.GetFlagString(c, models.HardwareFlagName, false, "Worker Node Isolation (shared/dedicated)", models.IsolationPublic)
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

	disableEncryptionStr, err := cliutils.GetFlagString(c, models.DisableDiskEncryptionFlagName, true, "Disable Disk encryption", strconv.FormatBool(infraConfig.DisableDiskEncryption))
	if err != nil {
		return err
	}
	disableEncryption, err := strconv.ParseBool(disableEncryptionStr)
	if err != nil {
		return err
	}
	encrypted := !disableEncryption

	labelsMap, err := cliutils.MakeLabelsMap(c)
	if err != nil {
		return err
	}

	workerPoolProperties := requests.VPCCreateWorkerPool{
		CommonCreateWorkerPool: requests.CommonCreateWorkerPool{
			Cluster:         clusterNameOrID,
			Name:            poolName,
			Flavor:          flavor,
			Isolation:       isolationType,
			DiskEncryption:  &encrypted,
			WorkerCount:     int(sizePerZone),
			Labels:          labelsMap,
			OperatingSystem: osName,
		},
		VPCCreateWorkerPoolFragment: requests.VPCCreateWorkerPoolFragment{
			VPCID: vpcID,
		},
		Zones: []requests.VPCCreateWorkerPoolZone{}, // zones created with 'zone add' command
	}

	endpoint := resources.GetArmadaV2Endpoint(c)

	for cl := 1; cl <= quantity; cl++ {
		if suffix || quantity > 1 {
			workerPoolProperties.Cluster = fmt.Sprintf("%s%d", clusterNameOrID, cl)
		}

		if err := endpoint.CreateWorkerPool(workerPoolProperties); err != nil {
			return err
		}

		fmt.Fprintf(c.App.Writer, "\nWorker Pool creation request for %s cluster '%s' successful.\n", provider, workerPoolProperties.Cluster)
	}

	return nil
}
