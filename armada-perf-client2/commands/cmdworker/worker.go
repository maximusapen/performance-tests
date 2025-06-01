/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2019, 2021 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmdworker

import (
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli"
	v2 "github.ibm.com/alchemy-containers/armada-api-model/json/v2"
	"github.ibm.com/alchemy-containers/armada-model/model"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/metrics"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

// RegisterCommands registers all `worker` commands
func RegisterCommands() []cli.Command {
	return []cli.Command{
		{
			Name:        models.NamespaceWorker,
			Description: "View and modify worker nodes for a cluster.",
			Category:    models.ClusterManagementCategory,
			Subcommands: []cli.Command{
				{
					Name:        models.CmdList,
					Description: "List all worker nodes in a cluster.",
					Flags: []cli.Flag{
						models.RequiredClusterFlag,
						models.PollIntervalFlag,
						models.TimeoutFlag,
						models.JSONOutFlag,
					},
					Action: WorkerList,
				},
				{
					Name:        models.CmdGet,
					Description: "View the details of a worker node.",
					Flags: []cli.Flag{
						models.RequiredClusterFlag,
						models.RequiredWorkerFlag,
						models.JSONOutFlag,
					},
					Action: workerGet,
				},
				{
					Name:        models.CmdReplace,
					Description: "Delete a worker node and replace it with a new worker node in the same worker pool.",
					Flags: []cli.Flag{
						models.RequiredClusterFlag,
						models.RequiredWorkerFlag,
						models.UpdateFlag,
					},
					Action: workerReplace,
				},
			},
		},
	}
}

// WorkerList will return all V2 properties that are common between classic and vpc workers
func WorkerList(c *cli.Context) error {
	var headerWritten bool
	var workerPollInterval time.Duration
	var timeout time.Duration

	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		fmt.Print(err.Error())
		return err
	}
	workerPollIntervalStr, err := cliutils.GetFlagString(c, models.PollIntervalFlagName, true, "Worker status polling interval", nil)
	if err != nil {
		return err
	}
	if len(workerPollIntervalStr) > 0 {
		workerPollInterval, err = time.ParseDuration(workerPollIntervalStr)
		if err != nil {
			return err
		}
	}

	timeoutStr, err := cliutils.GetFlagString(c, models.TimeoutFlagName, true, "Worker status polling timeout", nil)
	if err != nil {
		return err
	}
	if len(timeoutStr) > 0 {
		timeout, err = time.ParseDuration(timeoutStr)
		if err != nil {
			return err
		}
	}

	endpoint := resources.GetArmadaV2Endpoint(c)
	if workerPollInterval > 0 {
		var clusterComplete, masterFailed, operationTimeout bool
		var pollRequestTime, prevTime time.Time
		var v2Workers []v2.WorkerCommon
		var clusterWorkers = make(metrics.Workers)

		totalFailureCount := 0
		startTime := time.Now()

		maxFailures, err := cliutils.GetFlagInt(c, models.WorkerFailuresFlagName, true, "Maximum number of worker provision/deploy failure retries", nil)
		if err != nil {
			return fmt.Errorf("invalid '%s' specified: %s", models.WorkerFailuresFlagName, err.Error())
		}

		for !clusterComplete && !masterFailed {
			clusterComplete = true
			pollRequestTime = time.Now()

			_, v2Workers, err = endpoint.GetWorkers(clusterNameOrID)
			if err != nil || len(v2Workers) == 0 {
				// Error occurred or no workers returned yet, let's ignore and carry on polling
				time.Sleep(workerPollInterval)
				clusterComplete = false
				continue
			}

			// Need to handle master deployment failures to prevent infinite polling of workers
			_, v2Cluster, err := endpoint.GetCluster(clusterNameOrID)
			if err == nil {
				masterFailed = v2Cluster.MasterState() == *model.ClusterActualStateDeployFailed
			}

			if timeout > 0 && time.Since(startTime) > timeout {
				failedWorkers := make([]v2.WorkerCommon, 0)
				// Abort
				for _, worker := range v2Workers {
					workerCompleted := strings.EqualFold(worker.Health.State, *model.WorkerActualHealthStateNormal) || strings.EqualFold(worker.Health.State, *model.WorkerActualStateDeleted)
					if !workerCompleted {
						clusterWorkers[worker.ID].Failed = true

						health := v2.GetWorkerResponseHealth{State: worker.Lifecycle.ActualState, Message: "*** Operation Timed Out ***"}
						failedWorker := v2.WorkerCommon{ID: worker.ID, Health: health}
						failedWorkers = append(failedWorkers, failedWorker)
					}
				}
				v2Workers = failedWorkers
				operationTimeout = true
				break
			}

			headerWritten = false
			var workerStateCounts = make(map[string]int)

			for _, worker := range v2Workers {
				reportChange := false

				workerStateCounts[worker.Lifecycle.ActualState]++

				// Dirty, nasty, horrible code alert which I'm *still* disowning
				// We want to include time spent waiting for the master to be deployed in our state metrics.
				// Alas, this information is a worker 'message' field (rather than a worker state) so we have to check for a specific string (which can and has changed)
				// (or send all status transitions too, which given the fairly large number of them, we don't really want to do)
				const wfmd = "waiting_for_master_deployment"
				if worker.Lifecycle.Message == "Master is deploying" {
					worker.Lifecycle.ActualState = wfmd
				}

				// First time we've encountered this worker ?, if so, initialize it
				if ws, ok := clusterWorkers[worker.ID]; !ok {
					// Ignore any pre-existing workers
					// This will typically happen when called via worker-pool resize
					var ignoreWorker bool
					switch worker.Lifecycle.DesiredState {
					case *model.WorkerDesiredStateDeployed:
						ignoreWorker = strings.EqualFold(worker.Health.State, *model.WorkerActualHealthStateNormal)
					}
					if ignoreWorker {
						continue
					}

					reportChange = true
					clusterWorkers[worker.ID] = &metrics.Worker{
						CurState:  worker.Lifecycle.ActualState,
						CurStatus: worker.Lifecycle.Message,
						Durations: make(map[string]time.Duration)}
					ws = clusterWorkers[worker.ID]
					ws.Durations[worker.Lifecycle.ActualState] = 0
					clusterWorkers[worker.ID] = ws
				} else {
					// Worker moving to a new state ?
					if ws.CurState != worker.Lifecycle.ActualState {
						reportChange = true

						// Record the duration of the state we've just completed
						ws.Durations[ws.CurState] += pollRequestTime.Sub(prevTime)

						// Now update this worker to the new state
						ws.CurState = worker.Lifecycle.ActualState
						ws.CurStatus = worker.Lifecycle.Message
					} else {
						// Still in same state, update its total duration
						ws.Durations[ws.CurState] += pollRequestTime.Sub(prevTime)

						// Check the status too (for reporting updates to the user purposes only)
						if ws.CurStatus != worker.Lifecycle.Message {
							ws.CurStatus = worker.Lifecycle.Message
							reportChange = true
						}

					}
					clusterWorkers[worker.ID] = ws
				}

				workerState := worker.Lifecycle.ActualState
				switch workerState {
				case *model.WorkerActualStateDeployed, *model.WorkerActualStateDeleted:
					workerState = worker.Health.State
				}
				workerFailed := strings.EqualFold(worker.Lifecycle.ActualState, *model.WorkerActualStateDeployFailed) ||
					strings.EqualFold(worker.Lifecycle.ActualState, *model.WorkerActualStateProvisionFailed)

				// If this workers state has changed since last polling interval, then display update
				// (For display purposes, ensure we group all worker state changes under a single timestamp update)
				if reportChange {
					if !headerWritten {
						fmt.Fprintln(c.App.Writer, "\n", pollRequestTime.Format(time.Stamp))
						fmt.Fprintf(c.App.Writer, "\t%s\n", clusterNameOrID)
						headerWritten = true
					}
					fmt.Fprintf(c.App.Writer, "\t\t%s, %s, \"%s\"\n", worker.ID, worker.Lifecycle.ActualState, worker.Lifecycle.Message)
				}

				var workerCompleted bool
				switch worker.Lifecycle.DesiredState {
				case *model.WorkerDesiredStateDeployed:
					workerCompleted = strings.EqualFold(worker.Health.State, *model.WorkerActualHealthStateNormal)
				case *model.WorkerDesiredStateDeleted:
					workerCompleted = strings.EqualFold(worker.Health.State, *model.WorkerActualStateDeleted)
				}
				clusterComplete = clusterComplete && workerCompleted

				if workerFailed && !clusterWorkers[worker.ID].Failed {
					totalFailureCount++
					clusterWorkers[worker.ID].Failed = true

					if totalFailureCount > maxFailures {
						return fmt.Errorf("creation of worker \"%s\" failed. Maximum retries exceeded", worker.ID)
					}

					// Let's try a worker replace to fix this worker
					fmt.Fprintf(c.App.Writer, "\t\t\t*** Replacing failed worker \"%s\". Total failures: %d ***\n\n", worker.ID, totalFailureCount)
					if err := endpoint.ReplaceWorker(clusterNameOrID, worker.ID, false); err != nil {
						return err
					}
				}
			}

			// If we've had a worker state/status update, then update the number of workers currently in each state
			if headerWritten {
				for k, v := range workerStateCounts {
					fmt.Fprintf(c.App.Writer, "\t%s : %d/%d\n", k, v, len(v2Workers))
				}
				fmt.Fprintln(c.App.Writer)
			}

			prevTime = pollRequestTime

			if !clusterComplete && !masterFailed {
				time.Sleep(workerPollInterval)
			}
		}

		// Store metrics results in our metadata
		metricsData := cliutils.GetMetadataValue(c, models.MetricsFlagName).(*metrics.Data)
		metricsData.W = clusterWorkers

		// Cluster is complete, report final status - hopefully "normal" state
		fmt.Fprintln(c.App.Writer, pollRequestTime.Format(time.Stamp))
		fmt.Fprintf(c.App.Writer, "\t%s\n", clusterNameOrID)
		for _, worker := range v2Workers {
			fmt.Fprintf(c.App.Writer, "\t\t%s, %s, \"%s\"\n", worker.ID, worker.Health.State, worker.Health.Message)
		}

		if masterFailed {
			return fmt.Errorf("master deployment has failed")
		}

		if operationTimeout {
			return fmt.Errorf("timeout waiting for all workers to be ready. Timeout: %v", timeout)
		}
	} else {
		rawJSON, v2Workers, err := endpoint.GetWorkers(clusterNameOrID)
		if err != nil {
			return err
		}

		jsonoutput := c.Bool(models.JSONOutFlagName)
		if jsonoutput {
			cliutils.WriteJSON(c, rawJSON)
			return nil
		}

		if !headerWritten {
			fmt.Fprintln(c.App.Writer)
			fmt.Fprintln(c.App.Writer, "ID\tFlavor\tLocation\tState\tStatus\tVersion")
			fmt.Fprintln(c.App.Writer, "--\t------\t--------\t-----\t------\t-------")

			headerWritten = true
		}

		for _, worker := range v2Workers {
			workerKubeVersion := worker.KubeVersion.Actual
			if len(workerKubeVersion) == 0 {
				workerKubeVersion = worker.KubeVersion.Desired
			}

			workerState := worker.Lifecycle.ActualState
			workerMessage := worker.Lifecycle.Message

			switch workerState {
			case *model.WorkerActualStateDeployed:
				workerMessage = worker.Health.Message
				workerState = worker.Health.State
			}

			fmt.Fprintf(c.App.Writer,
				"%s\t%s\t%s\t%s\t%s\t%s\n",
				worker.ID,
				worker.Flavor,
				worker.Location,
				workerState,
				workerMessage,
				workerKubeVersion,
			)
		}
	}
	return nil
}

func workerGet(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	workerID, err := cliutils.GetFlagString(c, models.WorkerFlagName, false, models.WorkerFlagValue, nil)
	if err != nil {
		return err
	}

	jsonoutput := c.Bool(models.JSONOutFlagName)

	endpoint := resources.GetArmadaV2Endpoint(c)
	rawJSON, v2Worker, err := endpoint.GetWorker(clusterNameOrID, workerID)
	if err != nil {
		return err
	}

	if jsonoutput {
		cliutils.WriteJSON(c, rawJSON)
		return nil
	}

	fmt.Fprintln(c.App.Writer)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "ID:", v2Worker.ID)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Worker Pool ID:", v2Worker.PoolID)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Worker Pool Name:", v2Worker.PoolName)

	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Flavor:", v2Worker.Flavor)
	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Version:", v2Worker.KubeVersion.Actual)

	if v2Worker.KubeVersion.Actual != v2Worker.KubeVersion.Desired {
		fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Pending Version:", v2Worker.KubeVersion.Desired)
	}

	fmt.Fprintf(c.App.Writer, "%s\t%s\n", "Location:", v2Worker.Location)

	fmt.Fprintf(c.App.Writer, "%s\n\t%s\t%s\n\t%s\t%s\n", "Health:", "State:", v2Worker.Health.State, "Message:", v2Worker.Health.Message)
	fmt.Fprintf(c.App.Writer, "%s\n\t%s\t%s\n\t%s\t%s\n", "Lifecycle:", "State:", v2Worker.Lifecycle.ActualState, "Message:", v2Worker.Lifecycle.Message)

	return nil
}

// WorkerReplace will delete a worker node and replace it with a new worker node in the same worker pool
func workerReplace(c *cli.Context) error {
	clusterNameOrID, err := cliutils.GetFlagString(c, models.ClusterFlagName, false, models.ClusterFlagValue, nil)
	if err != nil {
		return err
	}

	workerID, err := cliutils.GetFlagString(c, models.WorkerFlagName, false, models.WorkerFlagValue, nil)
	if err != nil {
		return err
	}

	update := c.Bool(models.UpdateFlagName)

	endpoint := resources.GetArmadaV2Endpoint(c)

	err = endpoint.ReplaceWorker(clusterNameOrID, workerID, update)
	if err != nil {
		return err
	}

	fmt.Fprintf(c.App.Writer, "\nWorker replacement request for worker '%s' successful.\n", workerID)

	return nil
}
