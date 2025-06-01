/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2019, 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/registration"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/metrics"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
)

func main() {
	conf := cliutils.GetArmadaConfig()

	app := cli.NewApp()
	app.Name = "armada-perf-client2"
	app.Description = "Armada Performance Client Application"
	app.Version = "v2.0.0"

	tw := cliutils.NewWriter()
	app.Writer = tw

	app.Metadata = map[string]interface{}{
		"iamEndpoint":  conf.IBMCloud.IAMEndpoint,
		"accessToken":  conf.IBMCloud.AccessToken,
		"refreshToken": conf.IBMCloud.RefreshToken,

		"accountID":        conf.IBMCloud.AccountID,
		"apiKey":           conf.IBMCloud.APIKey,      // pragma: allowlist secret
		"infraApiKey":      conf.IBMCloud.InfraAPIKey, // pragma: allowlist secret
		"infraIamEndpoint": conf.IBMCloud.InfraIAMEndpoint,

		"iksEndpoint":     conf.IKS.Endpoint,
		"satlinkEndpoint": conf.SatLink.Endpoint,

		models.MetricsFlagName: new(metrics.Data),
	}

	app.Before = metrics.Initialize
	app.Commands = registration.CLICommands()
	app.After = metrics.WriteMetrics

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		cli.OsExiter(1)
	}

	tw.Flush()
}
