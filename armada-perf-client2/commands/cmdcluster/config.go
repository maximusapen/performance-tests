
package cmdcluster

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

// Handles 'cluster config' command
func clusterConfigAction(c *cli.Context) error {
	var network bool

	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}
	admin := c.Bool(models.AdminFlagName)

	// Network flag requires the admin endpoint.
	if admin {
		network = c.Bool(models.NetworkFlagName)
	}

	endpoint := resources.GetArmadaEndpoint(c)
	resp, err := endpoint.GetClusterConfig(clusterNameOrID, admin, network)
	if err != nil {
		return err
	}

	var filename string
	if admin {
		filename = fmt.Sprintf("kubeConfig%s-%s.zip", strings.Title(models.AdminFlagName), clusterNameOrID)
	} else {
		filename = fmt.Sprintf("kubeConfig-%s.zip", clusterNameOrID)
	}
	ioutil.WriteFile(filename, resp.Bytes(), 0644)
	return nil
}
