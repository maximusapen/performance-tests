/*******************************************************************************
 * I
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2019 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdkubeversions

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

// RegisterCommands registers the "versions" command
func RegisterCommands() []cli.Command {
	return []cli.Command{
		cli.Command{
			Name:        "versions",
			Description: "List the Kubernetes versions currently supported by IBM Cloud.",
			Category:    models.InformationalCategory,
			Flags: []cli.Flag{
				models.JSONOutFlag,
			},
			Action: versions,
		},
	}
}

func versions(c *cli.Context) error {
	endpoint := resources.GetArmadaEndpoint(c)

	allVersions, err := endpoint.GetVersions()
	if err != nil {
		return err
	}

	jsonoutput := c.Bool(models.JSONOutFlagName)
	if jsonoutput {
		cliutils.WriteJSON(c, allVersions)
		return nil
	}

	for flavor := range allVersions {
		fmt.Fprintln(c.App.Writer)
		fmt.Fprintln(c.App.Writer, strings.Title(flavor))
		for i := 0; i < len(flavor); i++ {
			fmt.Fprint(c.App.Writer, "-")
		}
		fmt.Fprintln(c.App.Writer)
		for _, v := range allVersions[flavor] {
			fmt.Fprintln(c.App.Writer, v)
		}
	}

	return nil
}
