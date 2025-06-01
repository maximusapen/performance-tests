/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2020, 2021 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdzone

import (
	"fmt"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-api-model/json/v2/vpc"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

func vpcClassicZoneAdd(c *cli.Context) error {
	return vpcZoneAdd(c, cliutils.GetInfrastructureConfig().VPCClassic)
}
func vpcNextGenZoneAdd(c *cli.Context) error {
	return vpcZoneAdd(c, cliutils.GetInfrastructureConfig().VPC)
}

// vpcZoneAdd will add a zone to a worker pool in a VPC cluster
func vpcZoneAdd(c *cli.Context, infraConfig *cliutils.TomlVPCInfrastructureConfig) error {
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

	subnetID, err := cliutils.GetFlagString(c, models.SubnetIDFlagName, true, "Virtual Private Cloud Subnet Identifier", infraConfig.Locations[zone].SubnetID)
	if err != nil {
		return err
	}

	zoneRequest := vpc.CreateWorkerpoolZoneRequest{
		Zone: vpc.Zone{
			ID:       zone,
			SubnetID: subnetID,
		},
		Cluster:    clusterNameOrID,
		WorkerPool: workerPool,
	}

	endpoint := resources.GetArmadaV2Endpoint(c)

	for cl := 1; cl <= quantity; cl++ {
		if suffix || quantity > 1 {
			zoneRequest.Cluster = fmt.Sprintf("%s%d", clusterNameOrID, cl)
		}

		if err = endpoint.AddZone(zoneRequest); err != nil {
			return err
		}

		fmt.Fprintf(c.App.Writer, "\nAddition of zone '%s' request for VPC cluster '%s' successful.\n", zone, zoneRequest.Cluster)
	}

	return nil
}
