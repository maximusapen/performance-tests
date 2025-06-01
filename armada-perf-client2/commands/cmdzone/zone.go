/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2020, 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdzone

import (
	"fmt"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

// RegisterCommands registers all `zone` commands
func RegisterCommands() []cli.Command {
	return []cli.Command{
		{
			Name:        models.NamespaceZone,
			Description: "List availability zones and modify the zones attached to a worker pool.",
			Category:    models.ClusterManagementCategory,
			Subcommands: []cli.Command{
				{
					Name:        models.CmdAdd,
					Description: "Add a zone to a worker pool in a cluster.",
					Subcommands: []cli.Command{
						{
							Name:        models.ProviderClassic,
							Description: "Add a zone to a worker pool in a classic cluster.",
							Flags: []cli.Flag{
								models.RequiredClusterFlag,
								models.SuffixFlag,
								models.QuantityFlag,
								models.RequiredZoneFlag,
								models.RequiredWorkerPoolFlag,
							},
							Action: zoneAdd,
						},
						{
							Name:        models.ProviderVPCClassic,
							Description: "Add a zone to a worker pool in a VPC-Classic (Gen 1) cluster.",
							Flags: []cli.Flag{
								models.RequiredClusterFlag,
								models.SuffixFlag,
								models.QuantityFlag,
								models.RequiredZoneFlag,
								models.RequiredWorkerPoolFlag,
								models.RequiredSubnetIDFlag,
							},
							Action: vpcClassicZoneAdd,
						},
						{
							Name:        models.ProviderVPCGen2,
							Description: "Add a zone to a worker pool in a VPC-Gen2 cluster.",
							Flags: []cli.Flag{
								models.RequiredClusterFlag,
								models.SuffixFlag,
								models.QuantityFlag,
								models.RequiredZoneFlag,
								models.RequiredWorkerPoolFlag,
								models.RequiredSubnetIDFlag,
							},
							Action: vpcNextGenZoneAdd,
						},
						{
							Name:        models.ProviderSatellite,
							Description: "Add a zone to a worker pool in a Satellite cluster.",
							Flags: []cli.Flag{
								models.RequiredClusterFlag,
								models.SuffixFlag,
								models.QuantityFlag,
								models.RequiredZoneFlag,
								models.RequiredWorkerPoolFlag,
							},
							Action: satelliteZoneAdd,
						}},
				},
				{
					Name:        models.CmdRemove,
					Description: "Remove a zone from a worker pool in a cluster.",
					Flags: []cli.Flag{
						models.RequiredClusterFlag,
						models.RequiredWorkerPoolFlag,
						models.RequiredZoneFlag,
					},
					Action: zoneRemove,
				},
			},
		},
	}
}

// zoneRemove will remove a zone from a worker pool
func zoneRemove(c *cli.Context) error {
	zone, err := cliutils.GetFlagString(c, models.ZoneFlagName, false, models.ZoneFlagValue, nil)
	if err != nil {
		return err
	}

	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	workerPoolNameOrID, err := cliutils.GetFlagString(c, models.WorkerPoolFlagName, false, models.WorkerPoolFlagValue, nil)
	if err != nil {
		return err
	}

	/* Looks like a bug in armada-api which prevents the v2 api finding the zone, use the v1 api for now
	https://github.ibm.com/alchemy-containers/armada-ironsides/issues/3684
	endpoint := resources.GetArmadaV2Endpoint(c)
	cfg := requests.RemoveWorkerPoolZoneReq{
		Cluster:    clusterNameOrID,
		WorkerPool: workerPoolNameOrID,
		Zone:       zone,
	}

	if err := endpoint.RemoveWorkerPoolZone(cfg); err != nil {
		return err
	}*/
	endpoint := resources.GetArmadaEndpoint(c)
	if err := endpoint.RemoveZone(clusterNameOrID, workerPoolNameOrID, zone); err != nil {
		return err
	}

	fmt.Fprintf(c.App.Writer, "Request to remove zone '%s' for cluster '%s' and worker-pool '%s' successful.\n", zone, clusterNameOrID, workerPoolNameOrID)

	return nil
}
