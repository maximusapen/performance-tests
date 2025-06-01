/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdmonconfig

import (
	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
)

// SubcommandRegister registers logging config command
func SubcommandRegister() cli.Command {
	return cli.Command{
		Name:        models.CmdConfig,
		Description: "Create a Sysdig monitoring configuration for a cluster.",
		Category:    models.MonitoringCategory,
		Subcommands: []cli.Command{
			{
				Name:        models.CmdCreate,
				Description: "Create a Sysdig monitoring configuration for a cluster.",
				Category:    models.MonitoringCategory,
				Flags: []cli.Flag{
					models.RequiredClusterFlag,
					models.HostFlag,
				},
				Action: createConfig,
			},
			{
				Name:        models.CmdList,
				Description: "List existing Sysdig monitoring configurations for a cluster.",
				Category:    models.MonitoringCategory,
				Flags: []cli.Flag{
					models.RequiredClusterFlag,
					models.HostFlag,
				},
				Action: listConfig,
			},
		},
	}
}

func createConfig(c *cli.Context) error {
	return nil
}

func listConfig(c *cli.Context) error {
	return nil
}
