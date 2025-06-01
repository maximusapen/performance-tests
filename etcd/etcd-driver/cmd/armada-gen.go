/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2021 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmd

import (
	"github.com/spf13/cobra"
	v3 "go.etcd.io/etcd/client/v3"
)

var armadaCmd = &cobra.Command{
	Use:   "armada",
	Short: "Armada key/value generator",
	Long:  "Generate etcd key/value pairs using the Armada schema",

	Run: armadaGenFunc,
}

var (
	armadaClusterCnt int
	armadaMasterCnt  int
	armadaRegionCnt  int
	armadaWorkerCnt  int
	cruisers         int
)

func init() {
	RootCmd.AddCommand(armadaCmd)
	armadaCmd.Flags().IntVar(&armadaRegionCnt, "regions", 4, "Total number of regions (max is 4)")
	armadaCmd.Flags().IntVar(&armadaClusterCnt, "clusters", 4, "Total number of clusters per region")
	armadaCmd.Flags().IntVar(&armadaWorkerCnt, "workers", 5, "Total number of workers per cluster")
	armadaCmd.Flags().IntVar(&armadaMasterCnt, "masters", 4, "Total number of masters per cluster")
}

func armadaGenFunc(cmd *cobra.Command, args []string) {

	setupCsvFile()

	mods := make(map[string]int, 4)
	mods["clusterid"] = armadaClusterCnt
	mods["masterid"] = armadaMasterCnt
	mods["region"] = armadaRegionCnt
	mods["workerid"] = armadaWorkerCnt

	armada := NewArmadaEngine(mods)

	totalConns := uint(1)
	totalClients := uint(1)

	clients := mustCreateClients(totalClients, totalConns)
	requests := make(chan v3.Op, totalClients)

	for i := range clients {
		wg.Add(1)
		go armada.doOps(clients[i], requests, i, false)
	}

	armada.GenerateKeys(0, requests, 0)
	wg.Wait()
}
