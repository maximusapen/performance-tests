/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2019, 2021 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdvpc

import (
	"fmt"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

// RegisterCommands registers the VPC list command
func RegisterCommands() []cli.Command {
	providerIDFlag := cli.StringFlag{
		Name:  models.ProviderIDFlagName,
		Usage: "The infrastructure provider type ID for the VPC. The default value is 'vpc-classic'.",
	}

	return []cli.Command{
		{
			Name:        "vpcs",
			Usage:       "List all VPCs",
			Description: "List all VPCs in the targeted resource group. If no resource group is targeted, then all VPCs in the account are listed.",
			Flags: []cli.Flag{
				providerIDFlag,
				models.JSONOutFlag,
			},
			Action:   getVPCs,
			Category: models.AccountManagementCategory,
		},
	}
}

func getVPCs(c *cli.Context) error {
	jsonoutput := c.Bool(models.JSONOutFlagName)

	providerID, err := cliutils.GetFlagString(c, models.ProviderIDFlagName, true, "", models.ProviderVPCGen2)
	if err != nil {
		return err
	}

	endpoint := resources.GetArmadaV2Endpoint(c)
	vpcs, err := endpoint.GetVPCs(providerID)
	if err != nil {
		return err
	}

	if jsonoutput {
		cliutils.WriteJSON(c, vpcs)
		return nil
	}

	fmt.Fprintln(c.App.Writer)
	fmt.Fprintln(c.App.Writer, "Name\tID")
	fmt.Fprintln(c.App.Writer, "----\t--")
	for _, v := range vpcs {
		fmt.Fprintf(c.App.Writer, "%s\t%s\n", v.Name, v.ID)
	}

	return nil
}
