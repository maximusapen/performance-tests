/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2019, 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package registration

import (
	"github.com/urfave/cli"

	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdalb"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdcluster"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdendpoint"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdhost"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdkubeversions"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdlocation"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdlogging"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdmonitoring"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdnlbdns"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdsubnets"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdvpc"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdworker"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdworkerpool"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdzone"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
)

// CLICommands registers commands
func CLICommands() []cli.Command {
	var commands []cli.Command
	commands = append(commands, cmdalb.RegisterCommands()...)
	commands = append(commands, cmdvpc.RegisterCommands()...)
	commands = append(commands, cmdcluster.RegisterCommands()...)
	commands = append(commands, cmdworker.RegisterCommands()...)
	commands = append(commands, cmdkubeversions.RegisterCommands()...)
	commands = append(commands, cmdnlbdns.RegisterCommands()...)
	commands = append(commands, cmdworkerpool.RegisterCommands()...)
	commands = append(commands, cmdsubnets.RegisterCommands()...)
	commands = append(commands, cmdzone.RegisterCommands()...)

	var satSubCommands []cli.Command
	satSubCommands = append(satSubCommands, cmdhost.RegisterCommands()...)
	satSubCommands = append(satSubCommands, cmdlocation.RegisterCommands()...)
	satSubCommands = append(satSubCommands, cmdendpoint.RegisterCommands()...)

	satCommands := []cli.Command{
		{Name: models.NamespaceSatellite,
			Description: "Manage IBM Cloud Satellite clusters.",
			Subcommands: satSubCommands,
			Category:    models.SatelliteCategory,
		},
	}

	var obSubCommands []cli.Command
	obSubCommands = append(obSubCommands, cmdlogging.RegisterCommands())
	obSubCommands = append(obSubCommands, cmdmonitoring.RegisterCommands())

	obCommands := []cli.Command{
		{Name: models.NamespaceObservability,
			Description: "Manage logging and monitoring configurations.",
			Subcommands: obSubCommands,
			Category:    models.ObservabilityCategory,
		},
	}

	commands = append(commands, satCommands...)
	commands = append(commands, obCommands...)

	return commands
}
