/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdnlbdns

import (
	"fmt"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-dns-model/v2/vpc"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

func vpcNlbdnsRemove(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	nlbSubdomain, err := cliutils.GetFlagString(c, models.DNSSubdomainFlagName, false, "nlb-dns Subdomain", nil)
	if err != nil {
		return err
	}

	if c.String(models.IPFlagName) != "" {
		return nlbDNSDel(c, nlbSubdomain)
	}

	// removing lb hostname
	newNlbVPCConfig := vpc.NlbVPCConfig{
		Cluster:      clusterNameOrID,
		NlbSubdomain: nlbSubdomain,
	}

	fmt.Fprintf(c.App.Writer, "Removing load balancer host name '%s'....\n", nlbSubdomain)

	endpoint := resources.GetArmadaV2Endpoint(c)
	if err := endpoint.RemoveLBHostname(newNlbVPCConfig); err != nil {
		return err
	}

	return nil
}
