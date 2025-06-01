/*******************************************************************************
 * I
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdlogconfig

import (
	"fmt"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"

	"github.ibm.com/alchemy-containers/armada-kedge-model/model"
)

// SubcommandRegister registers logging config command
func SubcommandRegister() cli.Command {
	return cli.Command{
		Name:        models.CmdConfig,
		Description: "Configure logging for the specified cluster.",
		Category:    models.LoggingCategory,
		Subcommands: []cli.Command{
			{
				Name:        models.CmdCreate,
				Description: "Create a LogDNA logging configuration for a cluster.",
				Category:    models.LoggingCategory,
				Flags: []cli.Flag{
					models.RequiredClusterFlag,
					models.RequiredLoggingInstanceFlag,
					models.LoggingKeyFlag,
				},
				Action: createConfig,
			},
			{
				Name:        models.CmdList,
				Description: "List existing LogDNA logging configurations for a cluster.",
				Category:    models.LoggingCategory,
				Flags: []cli.Flag{
					models.RequiredClusterFlag,
				},
				Action: listConfig,
			},
		},
	}
}

func createConfig(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	instance, err := cliutils.GetFlagString(c, models.InstanceFlagName, false, "", nil)
	if err != nil {
		return err
	}

	ingestionKey, err := cliutils.GetFlagString(c, models.LoggingKeyFlagName, true, "", nil)
	if err != nil {
		return err
	}

	endpoint := resources.GetArmadaV2Endpoint(c)

	if err := endpoint.OBCreateConfig(model.ServiceTypeLogging, clusterNameOrID, ingestionKey, instance, false); err != nil {
		return err
	}
	return nil
}

func listConfig(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	endpoint := resources.GetArmadaV2Endpoint(c)

	instances, err := endpoint.OBListConfig(model.ServiceTypeLogging, clusterNameOrID)
	if err != nil {
		return err
	}

	fmt.Println(instances)
	return nil
}
