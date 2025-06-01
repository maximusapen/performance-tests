/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdmonitoring

import (
	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdmonitoring/cmdmonconfig"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
)

// RegisterCommands registers the Satellite host commands
func RegisterCommands() cli.Command {
	return cli.Command{
		Name:        models.CmdMonitoring,
		Description: "Manage Sysdig monitoring configurations for a cluster.",
		Category:    models.MonitoringCategory,
		Subcommands: []cli.Command{
			cmdmonconfig.SubcommandRegister(),
		},
	}
}
