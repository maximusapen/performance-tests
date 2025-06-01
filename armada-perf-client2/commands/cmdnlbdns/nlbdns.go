/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

// Package cmdnlbdns registers and executes 'apc2 nlb-dns ...' commands.
package cmdnlbdns

import (
	"fmt"
	"strings"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

// RegisterCommands registers the NLB-DNS commands
func RegisterCommands() []cli.Command {
	return []cli.Command{
		{
			Name:        models.NamespaceNLBDNS,
			Description: "Create and manage host names for network load balancer (NLB) IP addresses in a cluster and health check monitors for host names.",
			Category:    models.ClusterComponentsCategory,
			Subcommands: []cli.Command{
				{
					Name:        models.CmdList,
					Description: "List the registered NLB host names and IP addresses in a cluster.",
					Flags: []cli.Flag{
						models.RequiredClusterFlag,
						models.JSONOutFlag,
					},
					Action: nlbdnsList,
				},
				{
					Name:        models.CmdGet,
					Description: "View the details of a registered NLB host name in a cluster.",
					Flags: []cli.Flag{
						models.RequiredClusterFlag,
						models.RequiredDNSSubdomainFlag,
						models.JSONOutFlag,
					},
					Action: nlbdnsGet,
				},
				{
					Name:        models.CmdAdd,
					Description: "Add an NLB IP to an existing host name.",
					Flags: []cli.Flag{
						models.RequiredClusterFlag,
						models.RequiredDNSIPFlag,
						models.RequiredNLBHostFlag,
						models.JSONOutFlag,
					},
					Action: nlbdnsAdd,
				},
				{
					Name:        models.CmdRemove,
					Description: "Remove an NLB IP or load balancer host name from an NLB host name.",
					Subcommands: []cli.Command{
						{
							Name:        models.ProviderClassic,
							Description: "Remove an NLB IP address from an NLB host name. If you remove all IPs from a host name, the host name still exists but no IPs are associated with it.",
							Flags: []cli.Flag{
								models.RequiredClusterFlag,
								models.RequiredDNSIPFlag,
								models.RequiredNLBHostFlag,
								models.JSONOutFlag,
							},
							Action: classicNlbdnsRemove,
						},
						{
							Name:        models.ProviderVPCGen2,
							Description: "Remove a load balancer host name or IP address from a DNS record in a VPC cluster.",
							Flags: []cli.Flag{
								models.RequiredClusterFlag,
								models.RequiredDNSSubdomainFlag,
								models.RequiredDNSIPFlag,
								models.JSONOutFlag,
							},
							Action: vpcNlbdnsRemove,
						},
					},
				},
			},
		},
	}
}

func nlbdnsList(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	endpoint := resources.GetArmadaV2Endpoint(c)
	rawJSON, classicClusterNLBs, vpcClusterNLB, err := endpoint.GetNlbDNSList(clusterNameOrID)
	if err != nil {
		return err
	}

	jsonoutput := c.Bool(models.JSONOutFlagName)
	if jsonoutput {
		cliutils.WriteJSON(c, rawJSON)
		return nil
	}

	// If Vlan ID is empty this is a VPC cluster response
	if len(classicClusterNLBs) >= 1 && classicClusterNLBs[0].NlbIPArray != nil {
		fmt.Fprintln(c.App.Writer)
		fmt.Fprintln(c.App.Writer, "Hostname\tIP(s)\tHealth Monitor\tSSL Cert Status\tSSL Cert Secret Name\tSecret Namespace\tStatus")
		fmt.Fprintln(c.App.Writer, "--------\t-----\t--------------\t---------------\t--------------------\t----------------\t------")

		for _, nlb := range classicClusterNLBs {
			fmt.Fprintf(c.App.Writer,
				"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				nlb.NlbHost,
				strings.Join(nlb.NlbIPArray, ","),
				nlb.NlbMonitorState,
				nlb.NlbSslSecretStatus,
				nlb.NlbSSLSecretName,
				nlb.SecretNamespace,
				nlb.NlbStatusMessage,
			)
		}
	} else {
		// VPC Cluster output
		fmt.Fprintln(c.App.Writer)
		fmt.Fprintln(c.App.Writer, "Subdomain\tTarget(s)\tSSL Cert Status\tSSL Cert Secret Name\tSecret Namespace\tStatus")
		fmt.Fprintln(c.App.Writer, "---------\t---------\t---------------\t--------------------\t----------------\t------")

		for _, nlbListConfig := range vpcClusterNLB {
			var targets string
			if len(nlbListConfig.Nlb.NlbIPArray) == 0 {
				targets = nlbListConfig.Nlb.LBHostname
			} else {
				targets = strings.Join(nlbListConfig.Nlb.NlbIPArray, ",")
			}

			fmt.Fprintf(c.App.Writer,
				"%s\t%s\t%s\t%s\t%s\t%s\n",
				nlbListConfig.Nlb.NlbSubdomain,
				targets,
				nlbListConfig.SecretStatus,
				nlbListConfig.SecretName,
				nlbListConfig.Nlb.SecretNamespace,
				nlbListConfig.Nlb.StatusMessage)
		}
	}

	return nil
}

func nlbdnsGet(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	nlbDNSSubdomain, err := cliutils.GetFlagString(c, models.DNSSubdomainFlagName, false, "nlb-dns Subdomain", nil)
	if err != nil {
		return err
	}
	nlbDNSSubdomain = strings.ToLower(nlbDNSSubdomain)

	endpoint := resources.GetArmadaV2Endpoint(c)
	rawJSON, classicClusterNLB, vpcClusterNLB, err := endpoint.GetNlbDNSDetails(clusterNameOrID, nlbDNSSubdomain)
	if err != nil {
		return err
	}

	jsonoutput := c.Bool(models.JSONOutFlagName)
	if jsonoutput {
		cliutils.WriteJSON(c, rawJSON)
		return nil
	}

	// Classic cluster response
	if classicClusterNLB.NlbIPArray != nil {
		fmt.Fprintln(c.App.Writer)
		fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Cluster:", classicClusterNLB.ClusterID)
		fmt.Fprintf(c.App.Writer, "%s\t%s\n", "NLB Subdomain:", classicClusterNLB.NlbHost)
		fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Target(s):", strings.Join(classicClusterNLB.NlbIPArray, ","))
		fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Status:", classicClusterNLB.NlbStatusMessage)
		fmt.Fprintln(c.App.Writer)
		fmt.Fprintln(c.App.Writer, "SSL Cert")
		fmt.Fprintf(c.App.Writer, "\t%s\t%s\n", "Secret Name:", classicClusterNLB.NlbSSLSecretName)
		fmt.Fprintf(c.App.Writer, "\t%s\t%s\n", "Secret Namespace:", classicClusterNLB.SecretNamespace)
		fmt.Fprintf(c.App.Writer, "\t%s\t%s\n", "Status:", classicClusterNLB.NlbSslSecretStatus)
	} else {
		// VPC Cluster response
		fmt.Fprintln(c.App.Writer)
		fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Cluster:", vpcClusterNLB.Nlb.Cluster)
		fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Subdomain:", vpcClusterNLB.Nlb.NlbSubdomain)
		fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Target(s):", vpcClusterNLB.Nlb.LBHostname)
		fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Status:", vpcClusterNLB.Nlb.StatusMessage)
		fmt.Fprintln(c.App.Writer)
		fmt.Fprintln(c.App.Writer, "SSL Cert")
		fmt.Fprintf(c.App.Writer, "\t%s\t%s\n", "Secret Name:", vpcClusterNLB.SecretName)
		fmt.Fprintf(c.App.Writer, "\t%s\t%s\n", "Secret Namespace:", vpcClusterNLB.Nlb.SecretNamespace)
		fmt.Fprintf(c.App.Writer, "\t%s\t%s\n", "Status:", vpcClusterNLB.SecretStatus)
	}

	return nil
}

func nlbdnsAdd(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	ips, err := getNlbIPFlag(c)
	if err != nil {
		return err
	}

	nlbHost, err := cliutils.GetFlagString(c, models.NLBHostFlagName, false, "NLB DNS host name", nil)
	if err != nil {
		return err
	}
	nlbHost = strings.ToLower(nlbHost)

	// fail with usage if hostname not present
	if !strings.Contains(nlbHost, "containers.appdomain.cloud") {
		return cliutils.IncorrectUsageError(c, "You must specify a load balancer host name in the '--nlb-host' flag.")
	}

	fmt.Fprintf(c.App.Writer, "Adding IPs: %s to NLB host name '%s' in cluster '%s'. It might take a while for the changes to be applied.\n", strings.Join(ips, ", "), clusterNameOrID, nlbHost)

	endpoint := resources.GetArmadaEndpoint(c)

	err = endpoint.AddNlbDNSIP(clusterNameOrID, ips, nlbHost)
	if err != nil {
		return err
	}

	return nil
}

func getNlbIPFlag(c *cli.Context) (ips []string, error error) {
	ips = c.StringSlice(models.IPFlagName)

	// fail with usage if IP not present
	errMessageBadIP := "You must specify at least one NLB IP address"
	if len(ips) == 0 {
		return nil, cliutils.IncorrectUsageError(c, errMessageBadIP)
	}

	for _, ip := range ips {
		ipParts := strings.Split(ip, ".")
		if len(ipParts) != 4 {
			return nil, cliutils.IncorrectUsageError(c, errMessageBadIP)
		}
	}

	return ips, nil

}

func nlbDNSDel(c *cli.Context, nlbHost string) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	ip, err := cliutils.GetFlagString(c, models.IPFlagName, false, "NLB IP address", nil)
	if err != nil {
		return err
	}

	// fail with usage if IP not present
	ipParts := strings.Split(ip, ".")
	if len(ipParts) != 4 {
		return cliutils.IncorrectUsageError(c, "You must specify an NLB IP address in the '--ip' flag")
	}

	nlbHost = strings.ToLower(nlbHost)

	// fail with usage if hostname not present
	if !strings.Contains(nlbHost, "containers.appdomain.cloud") {
		return cliutils.IncorrectUsageError(c, "You must specify a load balancer host name in the '--nlb-host' flag.")
	}

	endpoint := resources.GetArmadaEndpoint(c)

	fmt.Fprintf(c.App.Writer, "Deleting IP: %s from NLB host name '%s' in cluster '%s'. It might take a few minutes for the changes to be applied.\n", ip, clusterNameOrID, nlbHost)

	err = endpoint.DeleteNlbDNSIP(clusterNameOrID, ip, nlbHost)
	if err != nil {
		return err
	}

	return nil
}
