/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2020, 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdworkerpool

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-api-model/json/v2/requests"
	"github.ibm.com/alchemy-containers/armada-model/model"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdworker"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/metrics"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

// RegisterCommands registers `worker-pool` commands
func RegisterCommands() []cli.Command {
	return []cli.Command{
		{
			Name:        models.NamespaceWorkerPool,
			Description: "View and modify worker pools for a cluster.",
			Category:    models.ClusterManagementCategory,
			Subcommands: []cli.Command{
				{
					Name:        models.CmdCreate,
					Description: "Add a worker pool to a cluster. No worker nodes are created until zones are added to the worker pool.",
					Subcommands: []cli.Command{
						{
							Name:        models.ProviderClassic,
							Description: "Add a worker pool to a classic cluster. No worker nodes are created until zones are added to the worker pool.",
							Flags: []cli.Flag{
								models.RequiredWorkerPoolNameFlag,
								models.RequiredClusterFlag,
								models.SuffixFlag,
								models.QuantityFlag,
								models.SizePerZoneFlag,
								models.MachineTypeFlag,
								models.OperatingSystemFlag,
								models.LabelsFlag,
							},
							Action: classicWorkerPoolCreate,
						},
						{
							Name:        models.ProviderVPCClassic,
							Description: "Add a worker pool to a VPC-Classic (Gen 1) cluster. No worker nodes are created until zones are added to the worker pool.",
							Flags: []cli.Flag{
								models.RequiredWorkerPoolNameFlag,
								models.RequiredClusterFlag,
								models.SuffixFlag,
								models.QuantityFlag,
								models.SizePerZoneFlag,
								models.MachineTypeFlag,
								models.OperatingSystemFlag,
								models.RequiredVPCIDFlag,
								models.LabelsFlag,
							},
							Action: vpcClassicWorkerPoolCreate,
						},
						{
							Name:        models.ProviderVPCGen2,
							Description: "Add a worker pool to a VPC-Gen2 cluster. No worker nodes are created until zones are added to the worker pool.",
							Flags: []cli.Flag{
								models.RequiredWorkerPoolNameFlag,
								models.RequiredClusterFlag,
								models.SuffixFlag,
								models.QuantityFlag,
								models.SizePerZoneFlag,
								models.MachineTypeFlag,
								models.OperatingSystemFlag,
								models.RequiredVPCIDFlag,
								models.LabelsFlag,
							},
							Action: vpcNextGenWorkerPoolCreate,
						},
						{
							Name:        models.ProviderSatellite,
							Description: "Add a worker pool to an IBM Cloud Satellite cluster. No worker nodes are created until zones are added to the worker pool.",
							Flags: []cli.Flag{
								models.RequiredWorkerPoolNameFlag,
								models.RequiredClusterFlag,
								models.SuffixFlag,
								models.QuantityFlag,
								models.RequiredZoneFlag,
								models.SizePerZoneFlag,
								models.OperatingSystemFlag,
								models.LabelsFlag,
								models.HostLabelsFlag,
							},
							Action: satelliteWorkerPoolCreate,
						},
					},
				},
				{
					Name:        models.CmdList,
					Description: "List all worker pools in a cluster.",
					Flags: []cli.Flag{
						models.RequiredClusterFlag,
						models.JSONOutFlag,
					},
					Action: workerPoolList,
				},
				{
					Name:        models.CmdGet,
					Description: "View the details of a worker pool.",
					Flags: []cli.Flag{
						models.RequiredClusterFlag,
						models.RequiredWorkerPoolFlag,
						models.JSONOutFlag,
					},
					Action: workerPoolGet,
				},
				{
					Name:        models.CmdRemove,
					Description: "Remove a worker pool from a cluster.",
					Flags: []cli.Flag{
						models.RequiredClusterFlag,
						models.RequiredWorkerPoolFlag,
					},
					Action: workerPoolRemove,
				},
				{
					Name:        models.CmdResize,
					Description: "Resize the worker pool to the specified number of workers per zone.",
					Flags: []cli.Flag{
						models.RequiredClusterFlag,
						models.RequiredWorkerPoolFlag,
						models.RequiredSizePerZoneFlag,
						models.PollIntervalFlag,
						models.MetricsFlag,
					},
					Action: workerPoolResize,
				},
				{
					Name:        models.CmdRebalance,
					Description: "Rebalance a worker pool in a cluster. Rebalancing adds worker nodes so that the worker pool has the same number of nodes in each zone",
					Flags: []cli.Flag{
						models.RequiredClusterFlag,
						models.RequiredWorkerPoolFlag,
					},
					Action: workerPoolRebalance,
				},
				{
					Name:        models.NamespaceTaint,
					Description: "Set and remove Kubernetes taints for all worker nodes in a worker pool.",
					Subcommands: []cli.Command{
						{
							Name:        models.CmdSet,
							Description: "Set Kubernetes taints for all worker nodes in a worker pool. Taints prevent pods without matching tolerations from running on the worker nodes.",
							Flags: []cli.Flag{
								models.RequiredClusterFlag,
								models.RequiredWorkerPoolFlag,
								models.RequiredTaintFlag,
							},
							Action: workerPoolSetTaints,
						},
						{
							Name:        models.CmdRemove,
							Description: "Remove all Kubernetes taints from all worker nodes in a worker pool.",
							Flags: []cli.Flag{
								models.RequiredClusterFlag,
								models.RequiredWorkerPoolFlag,
							},
							Action: workerPoolRemoveTaints,
						},
					},
				},
				{
					Name:        models.NamespaceLabel,
					Description: "Set and remove custom Kubernetes labels for all worker nodes in a worker pool.",
					Subcommands: []cli.Command{
						{
							Name:        models.CmdSet,
							Description: "Set custom Kubernetes labels for all worker nodes in a worker pool.",
							Flags: []cli.Flag{
								models.RequiredClusterFlag,
								models.RequiredWorkerPoolFlag,
								models.RequiredLabelsFlag,
							},
							Action: workerPoolSetLabels,
						},
						{
							Name:        models.CmdRemove,
							Description: "Remove all custom Kubernetes labels from all worker nodes in a worker pool.",
							Flags: []cli.Flag{
								models.RequiredClusterFlag,
								models.RequiredWorkerPoolFlag,
							},
							Action: workerPoolRemoveLabels,
						},
					},
				},
			},
		},
	}
}

// workerPoolList handles a 'worker-pool ls' command
func workerPoolList(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	endpoint := resources.GetArmadaV2Endpoint(c)

	rawJSON, v2WorkerPools, err := endpoint.GetWorkerPools(clusterNameOrID)
	if err != nil {
		return err
	}

	jsonoutput := c.Bool(models.JSONOutFlagName)
	if jsonoutput {
		cliutils.WriteJSON(c, rawJSON)
		return nil
	}

	fmt.Fprintln(c.App.Writer)
	fmt.Fprintln(c.App.Writer, "Pool Name\tID\tFlavor\tWorkers\tProvider")
	fmt.Fprintln(c.App.Writer, "---------\t--\t------\t-------\t--------")

	for _, workerPool := range v2WorkerPools {
		fmt.Fprintf(c.App.Writer,
			"%s\t%s\t%s\t%d\t%s\n",
			workerPool.PoolName,
			workerPool.ID,
			workerPool.Flavor,
			workerPool.WorkersPerZone,
			workerPool.Provider,
		)
	}

	return nil
}

// workerPoolGet handles a 'worker-pool get' command
func workerPoolGet(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	workerPoolID, err := cliutils.GetFlagString(c, models.WorkerPoolFlagName, false, models.WorkerPoolFlagValue, nil)
	if err != nil {
		return err
	}

	jsonoutput := c.Bool(models.JSONOutFlagName)

	endpoint := resources.GetArmadaV2Endpoint(c)
	rawJSON, v2WorkerPool, err := endpoint.GetWorkerPool(clusterNameOrID, workerPoolID)
	if err != nil {
		return err
	}

	if jsonoutput {
		cliutils.WriteJSON(c, rawJSON)
		return nil
	}

	fmt.Fprintln(c.App.Writer)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Worker Pool Name:", v2WorkerPool.PoolName)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "ID:", v2WorkerPool.ID)

	wpState := v2WorkerPool.Lifecycle.ActualState
	if wpState == "" {
		wpState = v2WorkerPool.Lifecycle.DesiredState
	}
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "State:", wpState)

	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Isolation:", v2WorkerPool.Isolation)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Operating System:", v2WorkerPool.OperatingSystem)
	fmt.Fprintf(c.App.Writer, "%s\t%t\n", "Balanced:", v2WorkerPool.IsBalanced)
	fmt.Fprintf(c.App.Writer, "%s\t%t\n", "Autoscale Enabled:", v2WorkerPool.AutoscaleEnabled)
	fmt.Fprintf(c.App.Writer, "%s\t%d\n", "Workers per Zone:", v2WorkerPool.WorkersPerZone)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Labels:", v2WorkerPool.Labels)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Flavor:", v2WorkerPool.Flavor)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Provider:", v2WorkerPool.Provider)

	fmt.Fprintf(c.App.Writer, "Zones\n-----\nZone\tWorkers\tSubnets\n")
	for _, zone := range v2WorkerPool.Zones {
		var subnetList []string
		for _, s := range zone.Subnets {
			subnetList = append(subnetList, s.ID)
		}
		fmt.Fprintf(c.App.Writer, "%s\t%d\t%s\n", zone.ID, zone.WorkerCount, strings.Join(subnetList, ", "))
	}
	return nil
}

// workerPoolRemove handles a 'worker-pool rm' command
func workerPoolRemove(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	workerPoolNameOrID, err := cliutils.GetFlagString(c, models.WorkerPoolFlagName, false, models.WorkerPoolFlagValue, nil)
	if err != nil {
		return err
	}

	endpoint := resources.GetArmadaV2Endpoint(c)

	cfg := requests.RemoveWorkerPool{
		Cluster:    clusterNameOrID,
		WorkerPool: workerPoolNameOrID,
	}
	if err := endpoint.RemoveWorkerPool(cfg); err != nil {
		return err
	}

	fmt.Fprintf(c.App.Writer, "Worker-pool '%s' deletion request for cluster '%s' successful.\n", workerPoolNameOrID, clusterNameOrID)

	return nil
}

// workerPoolResize handles a 'worker-pool resize' command
func workerPoolResize(c *cli.Context) error {

	// Initialize metrics data
	metricsData := cliutils.GetMetadataValue(c, models.MetricsFlagName).(*metrics.Data)
	metricsData.Command = fmt.Sprintf("%s-worker-pool", strings.Split(cliutils.GetCommandName(c), " ")[1])

	metricsData.Enabled = cliutils.GetFlagBool(c, models.MetricsFlagName)

	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	workerPoolNameOrID, err := cliutils.GetFlagString(c, models.WorkerPoolFlagName, false, models.WorkerPoolFlagValue, nil)
	if err != nil {
		return err
	}

	newSizeString, err := cliutils.GetFlagString(c, models.SizePerZoneFlagName, false, "", nil)
	if err != nil {
		return err
	}
	newSize, err := strconv.ParseInt(newSizeString, 10, 0)
	if err != nil {
		return err
	}

	pollIntervalStr, err := cliutils.GetFlagString(c, models.PollIntervalFlagName, true, "Status polling interval", nil)
	if err != nil {
		return err
	}
	pollInterval, err := time.ParseDuration(pollIntervalStr)
	if err != nil && len(pollIntervalStr) > 0 {
		fmt.Fprintf(c.App.Writer, "Invalid --%s %s - %s. Ignoring\n", models.PollIntervalFlagName, pollIntervalStr, err.Error())
	}

	endpoint := resources.GetArmadaV2Endpoint(c)

	if metricsData.Enabled {
		_, v2Workers, err := endpoint.GetWorkers(clusterNameOrID)
		if err != nil {
			return err
		}

		currentSize := len(v2Workers)
		if int(newSize) > currentSize {
			metricsData.Command += "-up"
		} else if int(newSize) < currentSize {
			metricsData.Command += "-down"
		}
	}

	cfg := requests.ResizeWorkerPool{
		Cluster:                    clusterNameOrID,
		WorkerPool:                 workerPoolNameOrID,
		Size:                       int(newSize),
		AllowSingleOpenShiftWorker: false,
	}

	startTime := time.Now()
	if err := endpoint.ResizeWorkerPool(cfg); err != nil {
		return err
	}

	fmt.Fprintf(c.App.Writer, "Worker-pool '%s' resize request for cluster '%s' successful.\n", workerPoolNameOrID, clusterNameOrID)

	if pollInterval > 0 {
		// Wait for the resize request to be actioned before polling workers
		for {
			_, v2Workers, err := endpoint.GetWorkers(clusterNameOrID)
			if err != nil {
				return err
			}

			desiredWorkers := 0
			for _, w := range v2Workers {
				if w.Lifecycle.DesiredState == *model.WorkerDesiredStateDeployed {
					desiredWorkers++
				}
			}
			if desiredWorkers == int(newSize) {
				break
			}

			fmt.Fprintf(c.App.Writer, "Waiting for worker-pool resize request to be actioned.\n")
			time.Sleep(time.Second * 5)
		}

		c.Set(models.ClusterFlagName, clusterNameOrID) // Need to set the --cluster flag as required by "worker ls"
		if err := cmdworker.WorkerList(c); err != nil {
			return err
		}
	}

	totalTime := time.Since(startTime)
	metricsData.P = append(metricsData.P, totalTime)

	fmt.Fprintf(c.App.Writer, "Total Duration: %.1fs\n", totalTime.Seconds())

	return nil
}

// workerPoolResize handles a 'worker-pool rebalance' command
func workerPoolRebalance(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	workerPoolNameOrID, err := cliutils.GetFlagString(c, models.WorkerPoolFlagName, false, models.WorkerPoolFlagValue, nil)
	if err != nil {
		return err
	}

	endpoint := resources.GetArmadaV2Endpoint(c)

	cfg := requests.RebalanceWorkerPool{
		Cluster:    clusterNameOrID,
		WorkerPool: workerPoolNameOrID,
	}

	if err := endpoint.RebalanceWorkerPool(cfg); err != nil {
		return err
	}

	fmt.Fprintf(c.App.Writer, "Worker-pool '%s' rebalance request for cluster '%s' successful.\n", workerPoolNameOrID, clusterNameOrID)

	return nil
}

// workerPoolSetTaints sets the taints on a worker pool
func workerPoolSetTaints(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	workerPoolNameOrID, err := cliutils.GetFlagString(c, models.WorkerPoolFlagName, false, models.WorkerPoolFlagValue, nil)
	if err != nil {
		return err
	}

	taintsMap := make(map[string]string)
	taints := c.StringSlice(models.TaintFlagName)

	// fail with usage if no taints specified
	errMessageTaints := "You must specify at least one taint"
	if len(taints) == 0 {
		return cliutils.IncorrectUsageError(c, errMessageTaints)
	}

	for _, t := range taints {
		if t == "" {
			continue
		}
		labelTokens := strings.SplitN(t, "=", 2)
		if len(labelTokens) != 2 {
			return cliutils.IncorrectUsageError(c, fmt.Sprintf(cliutils.KeyValuePairInvalidMsg, t, models.TaintFlagFormat))
		}
		key, value := labelTokens[0], labelTokens[1]
		taintsMap[key] = value
	}

	endpoint := resources.GetArmadaV2Endpoint(c)
	err = endpoint.SetWorkerPoolTaints(clusterNameOrID, workerPoolNameOrID, taintsMap)
	if err != nil {
		fmt.Fprintf(c.App.Writer, "Error setting worker pool taints for cluster '%s'. Error: %s\n", clusterNameOrID, err.Error())
		return err
	}

	return nil
}

// workerPoolRemoveTaints removes the taints from a worker pool
func workerPoolRemoveTaints(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	workerPoolNameOrID, err := cliutils.GetFlagString(c, models.WorkerPoolFlagName, false, models.WorkerPoolFlagValue, nil)
	if err != nil {
		return err
	}

	taintsMap := make(map[string]string)
	endpoint := resources.GetArmadaV2Endpoint(c)
	err = endpoint.SetWorkerPoolTaints(clusterNameOrID, workerPoolNameOrID, taintsMap)
	if err != nil {
		fmt.Fprintf(c.App.Writer, "Error removing worker pool taints for cluster '%s'. Error: %s\n", clusterNameOrID, err.Error())
		return err
	}
	return nil
}

// workerPoolSetLabels sets the labels on a worker pool
func workerPoolSetLabels(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	workerPoolNameOrID, err := cliutils.GetFlagString(c, models.WorkerPoolFlagName, false, models.WorkerPoolFlagValue, nil)
	if err != nil {
		return err
	}

	labelsMap, err := cliutils.MakeLabelsMap(c)
	if err != nil {
		return err
	}
	if len(labelsMap) == 0 {
		failureMessage := fmt.Sprintf(cliutils.MissingParameterMsg, "FlagName", models.LabelFlagName)
		return cliutils.IncorrectUsageError(c, failureMessage)
	}

	endpoint := resources.GetArmadaV2Endpoint(c)
	err = endpoint.SetWorkerPoolLabels(clusterNameOrID, workerPoolNameOrID, labelsMap)
	if err != nil {
		fmt.Fprintf(c.App.Writer, "Error setting worker pool labels for cluster '%s'\n", clusterNameOrID)
		return err
	}
	return nil
}

// workerPoolRemoveLabels removes the labels from a worker pool
func workerPoolRemoveLabels(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	workerPoolNameOrID, err := cliutils.GetFlagString(c, models.WorkerPoolFlagName, false, models.WorkerPoolFlagValue, nil)
	if err != nil {
		return err
	}

	labelsMap := make(map[string]string)
	endpoint := resources.GetArmadaV2Endpoint(c)

	err = endpoint.SetWorkerPoolLabels(clusterNameOrID, workerPoolNameOrID, labelsMap)
	if err != nil {
		fmt.Fprintf(c.App.Writer, "Error removing custom worker pool labels for cluster '%s'. Error: %s\n", clusterNameOrID, err.Error())
		return err
	}

	return nil
}
