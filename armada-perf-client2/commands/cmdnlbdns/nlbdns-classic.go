/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdnlbdns

import (
	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
)

func classicNlbdnsRemove(c *cli.Context) error {
	nlbHost, err := cliutils.GetFlagString(c, models.NLBHostFlagName, false, "NLB DNS host name", nil)
	if err != nil {
		return err
	}
	return nlbDNSDel(c, nlbHost)
}
