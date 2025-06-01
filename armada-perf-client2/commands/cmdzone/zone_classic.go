/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2020 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdzone

import (
	"fmt"

	"github.com/urfave/cli"
	apiModelV1 "github.ibm.com/alchemy-containers/armada-model/model/api/json/v1"

	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

// zoneAdd will add a zone to a worker pool in a classic cluster
func zoneAdd(c *cli.Context) error {
	infraConfig := cliutils.GetInfrastructureConfig().Classic

	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	suffix := c.Bool(models.SuffixFlagName)

	quantity, err := cliutils.GetFlagInt(c, models.QuantityFlagName, true, "Number of clusters", nil)
	if err != nil {
		return err
	}

	workerPool, err := cliutils.GetFlagString(c, models.WorkerPoolFlagName, false, "Name of the worker pool to which the zone should be added.", nil)
	if err != nil {
		return err
	}

	zone, err := cliutils.GetFlagString(c, models.ZoneFlagName, false, "Name of the zone to be added.", nil)
	if err != nil {
		return err
	}

	vlanPrivate, err := cliutils.GetFlagString(c, models.PrivateVLANFlagName, false, "Private VLAN Name", infraConfig.PrivateVlan)
	if err != nil {
		return err
	}

	vlanPublic, err := cliutils.GetFlagString(c, models.PublicVLANFlagName, false, "Public VLAN Name", infraConfig.PublicVlan)
	if err != nil {
		return err
	}

	workerPoolZoneProperties := apiModelV1.WorkerPoolZone{
		ID: zone,
		WorkerPoolZoneNetwork: apiModelV1.WorkerPoolZoneNetwork{
			PrivateVLAN: vlanPrivate,
			PublicVLAN:  vlanPublic,
		},
	}

	endpoint := resources.GetArmadaEndpoint(c)

	for cl := 1; cl <= quantity; cl++ {
		clusterName := clusterNameOrID

		if suffix || quantity > 1 {
			clusterName = fmt.Sprintf("%s%d", clusterName, cl)
		}

		if err = endpoint.AddZone(clusterName, workerPool, workerPoolZoneProperties); err != nil {
			return err
		}

		fmt.Fprintf(c.App.Writer, "\nAddition of zone '%s' request for classic cluster '%s' successful.\n", zone, clusterName)
	}

	return nil
}
