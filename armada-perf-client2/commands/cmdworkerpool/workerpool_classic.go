/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2020, 2021 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdworkerpool

import (
	"fmt"
	"strconv"

	"github.com/urfave/cli"
	apiModelV1 "github.ibm.com/alchemy-containers/armada-model/model/api/json/v1"

	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

func classicWorkerPoolCreate(c *cli.Context) error {
	infraConfig := cliutils.GetInfrastructureConfig().Classic

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

	machineType, err := cliutils.GetFlagString(c, models.MachineTypeFlagName, false, "Worker Node Machine Flavor", infraConfig.Flavor)
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

	// Set the properties of the desired worker pool
	workerPoolProperties := apiModelV1.WorkerPoolRequest{
		WorkerPool: apiModelV1.WorkerPool{
			Name:            poolName,
			Size:            int(sizePerZone),
			MachineType:     machineType,
			Isolation:       isolationType,
			Labels:          labelsMap,
			OperatingSystem: osName,
		},
		DiskEncryption: encrypted,
	}

	endpoint := resources.GetArmadaEndpoint(c)

	for cl := 1; cl <= quantity; cl++ {
		clusterName := clusterNameOrID

		if suffix || quantity > 1 {
			clusterName = fmt.Sprintf("%s%d", clusterName, cl)
		}

		if err := endpoint.CreateWorkerPool(clusterName, workerPoolProperties); err != nil {
			return err
		}

		fmt.Fprintf(c.App.Writer, "\nWorker Pool creation request for classic cluster '%s' successful.\n", clusterNameOrID)
	}

	return nil
}
