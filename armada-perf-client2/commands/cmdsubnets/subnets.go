/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2021 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdsubnets

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"
	apiModelV1 "github.ibm.com/alchemy-containers/armada-model/model/api/json/v1"

	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

// VPCIDFlag is the flag used for specifying VPC Identifier
var VPCIDFlag = cli.StringFlag{
	Name:  models.VPCIDFlagName,
	Usage: "VPC-Gen2 Identifier. Required for VPC Gen2 provider types",
}

// RegisterCommands registers the `subnets` command
func RegisterCommands() []cli.Command {
	providerIDFlag := cli.StringFlag{
		Name:  models.ProviderIDFlagName,
		Usage: "Infrastructure provider. Available options: classic, vpc-gen2",
	}

	return []cli.Command{
		{
			Name:        "subnets",
			Usage:       "List available subnets in your IBM Cloud infrastructure account",
			Description: "List available subnets in your IBM Cloud infrastructure account.",
			Flags: []cli.Flag{
				providerIDFlag,
				VPCIDFlag,
				models.ZoneFlag,
				models.JSONOutFlag,
			},
			Action:   commandSubnets(getV1Subnets, getV2Subnets),
			Category: models.AccountManagementCategory,
		},
	}
}

func commandSubnets(
	getV1Subnets func(c *cli.Context) error,
	getV2Subnets func(*cli.Context) error,
) cli.ActionFunc {
	return func(c *cli.Context) error {
		// Get provider, default to classic
		provider := c.String(models.ProviderIDFlagName)
		if provider == "" {
			provider = models.ProviderClassic
		}

		if provider == models.ProviderClassic {
			return getV1Subnets(c)
		} else if provider == models.ProviderVPCGen2 {
			return getV2Subnets(c)
		}

		return fmt.Errorf("Unrecognised provider specified : '%s'", provider)
	}
}

func getV1Subnets(c *cli.Context) error {
	var subnets []apiModelV1.Subnet
	var err error
	var filteredSubnets []apiModelV1.Subnet
	jsonoutput := c.Bool(models.JSONOutFlagName)

	endpoint := resources.GetArmadaEndpoint(c)
	subnets, err = endpoint.GetSubnets()
	if err != nil {
		return err
	}

	// Filter primary subnets from api request since these subnets should
	// never be bound to a customers cluster
	for _, tempSubnet := range subnets {
		if !strings.Contains(tempSubnet.Properties.SubnetType, "primary") {
			filteredSubnets = append(filteredSubnets, tempSubnet)
		}
	}

	if jsonoutput {
		cliutils.WriteJSON(c, filteredSubnets)
		return nil
	}

	fmt.Fprintln(c.App.Writer)
	fmt.Fprintln(c.App.Writer, "ID\tNetwork\tVLAN ID\tType\tBound Cluster")
	fmt.Fprintln(c.App.Writer, "--\t-------\t-------\t----\t-------------")
	for _, s := range subnets {
		fmt.Fprintf(c.App.Writer, "%s\t%s\t%s\t%s\t%s\n", s.ID, s.Properties.DisplayLabel, s.VLANID, s.Type, s.Properties.BoundCluster)
	}

	return nil
}

func getV2Subnets(c *cli.Context) error {
	vpcID, err := cliutils.GetFlagString(c, models.VPCIDFlagName, false, "VPC identifier", nil)
	if err != nil {
		return err
	}
	zone, err := cliutils.GetFlagString(c, models.ZoneFlagName, false, "Zone", nil)
	if err != nil {
		return err
	}
	jsonoutput := c.Bool(models.JSONOutFlagName)

	endpoint := resources.GetArmadaV2Endpoint(c)
	subnets, err := endpoint.GetSubnets(vpcID, zone)
	if err != nil {
		return err
	}
	if jsonoutput {
		cliutils.WriteJSON(c, subnets)
		return nil
	}

	fmt.Fprintln(c.App.Writer)
	fmt.Fprintln(c.App.Writer, "Name\tID\tIPv4 CIDR Block")
	fmt.Fprintln(c.App.Writer, "----\t--\t---------------")
	for _, s := range subnets {
		fmt.Fprintf(c.App.Writer, "%s\t%s\t%s\n", s.Name, s.ID, s.IPv4CIDR)
	}

	return nil
}
