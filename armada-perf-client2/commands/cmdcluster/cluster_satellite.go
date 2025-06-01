
package cmdcluster

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-api-model/json/v2/requests"
	"github.ibm.com/alchemy-containers/armada-model/model"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/metrics"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

// satelliteClusterCreateTP implements a thread pool for handling parallel Satellite cluster creation
func satelliteClusterCreateTP(c *cli.Context, cwg *sync.WaitGroup, requests <-chan requests.MultishiftCreateCluster, results chan<- clusterResponse) {
	endpoint := resources.GetArmadaV2Endpoint(c)

	// Block until cluster is created and report worker status at the specified interval ?
	pollIntervalStr, _ := cliutils.GetFlagString(c, models.PollIntervalFlagName, true, "Cluster status polling interval", nil)
	pollInterval, err := time.ParseDuration(pollIntervalStr)
	if err != nil && len(pollIntervalStr) > 0 {
		fmt.Fprintf(c.App.Writer, "Invalid --%s %s - %s. Ignoring\n", models.PollIntervalFlagName, pollIntervalStr, err.Error())
	}

	timeoutStr, _ := cliutils.GetFlagString(c, models.TimeoutFlagName, true, "Satellite cluster timeout", nil)
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil && len(timeoutStr) > 0 {
		fmt.Fprintf(c.App.Writer, "Invalid --%s %s - %s. Ignoring\n", models.TimeoutFlagName, timeoutStr, err.Error())
	}

	for config := range requests {
		startTime := time.Now()
		clusterID, err := endpoint.CreateSatelliteCluster(config)
		if err != nil {
			results <- clusterResponse{err: err}
		} else {
			fmt.Fprintf(c.App.Writer, "\nCluster creation request for '%s' successful. ID: %s\n", config.Name, clusterID)
			fmt.Fprintln(c.App.Writer)

			if pollInterval > 0 {
				// Poll on master being ready
				var masterReady, masterFailed bool

				fmt.Fprintln(c.App.Writer, startTime.Format(time.Stamp))
				fmt.Fprintf(c.App.Writer, "\tWaiting for deployment of '%s' master to complete\n", config.Name)
				fmt.Fprintln(c.App.Writer)

				for !masterReady {
					if timeout > 0 && time.Since(startTime) > timeout {
						masterErr := fmt.Errorf("timeout waiting for Satellite cluster master to be ready. Timeout: %v", timeout)
						results <- clusterResponse{err: masterErr, duration: time.Since(startTime)}
					}

					if _, v2Cluster, err := endpoint.GetCluster(config.Name); err != nil {
						results <- clusterResponse{err: err}
						break
					} else {
						masterReady = v2Cluster.MasterStatus() == models.MasterReadyStatus
						masterFailed = v2Cluster.MasterState() == *model.ClusterActualStateDeployFailed
						if !masterReady && !masterFailed {
							time.Sleep(pollInterval)
						} else {
							var masterErr error
							if masterFailed {
								masterErr = fmt.Errorf("master deployment has failed")
							}
							results <- clusterResponse{err: masterErr, duration: time.Since(startTime)}
						}
					}
				}

				fmt.Fprintln(c.App.Writer)
			} else {
				results <- clusterResponse{err: nil, duration: time.Since(startTime)}
			}
		}
		cwg.Done()
	}
}

func satelliteClusterCreate(c *cli.Context) error {
	var cwg sync.WaitGroup

	// Initialize metrics data
	metricsData := cliutils.GetMetadataValue(c, models.MetricsFlagName).(*metrics.Data)
	metricsData.Command = fmt.Sprintf("%s-cluster", strings.Split(cliutils.GetCommandName(c), " ")[0])
	metricsData.Enabled = cliutils.GetFlagBool(c, models.MetricsFlagName)

	name, err := cliutils.GetFlagString(c, models.NameFlagName, false, "Name of the cluster to be created.", nil)
	if err != nil {
		return err
	}

	locationNameOrID, err := cliutils.GetFlagString(c, models.LocationFlagName, false, "Location name or ID", nil)
	if err != nil {
		return err
	}

	kubeVersion, err := cliutils.GetFlagString(c, models.KubeVersionFlagName, true, "Cluster Kubernetes version", nil)
	if err != nil {
		return err
	}

	zone, err := cliutils.GetFlagString(c, models.ZoneFlagName, true, "Zone for the default worker pool", nil)
	if err != nil {
		return err
	}

	// Flag to request that a numerical suffix is added to the specified cluster name
	// Note that if the specified quantity is more than one, a numerical suffix will be automatically added.
	suffix := cliutils.GetFlagBool(c, models.SuffixFlagName)

	quantity, err := cliutils.GetFlagInt(c, models.QuantityFlagName, true, "Number of clusters", nil)
	if err != nil {
		return err
	}

	threads, err := cliutils.GetFlagInt(c, models.ThreadFlagName, true, "Number of threads to handle multiple requests in parallel", nil)
	if err != nil {
		return err
	}

	osName, err := cliutils.GetFlagString(c, models.OperatingSystemFlagName, true, "Operating System Name", nil)
	if err != nil {
		return err
	}

	// Build the config object
	configTemplate := requests.MultishiftCreateCluster{
		MultishiftCreateClusterFragment: requests.MultishiftCreateClusterFragment{
			Controller: locationNameOrID,
		},
		Name:            name,
		KubeVersion:     kubeVersion,
		OperatingSystem: osName,
		Zone:            zone,
	}

	// Setup a pool of worker threads to handle cluster creation request(s)
	requests := make(chan requests.MultishiftCreateCluster, quantity)
	results := make(chan clusterResponse, quantity)
	cwg.Add(quantity)

	for t := 1; t <= threads; t++ {
		go satelliteClusterCreateTP(c, &cwg, requests, results)
	}

	startTime := time.Now()
	for cl := 1; cl <= quantity; cl++ {
		config := configTemplate

		if suffix || quantity > 1 {
			config.Name = fmt.Sprintf("%s%d", configTemplate.Name, cl)
		}

		// Send create cluster request to the thread pool
		requests <- config
	}
	close(requests)
	cwg.Wait()

	endTime := time.Since(startTime)
	fmt.Fprintf(c.App.Writer, "Total Duration: %.1fs\n", endTime.Seconds())

	for cl := 1; cl <= quantity; cl++ {
		r := <-results
		if r.err != nil {
			return r.err
		}
		metricsData.C.Duration = append(metricsData.C.Duration, r.duration)
	}

	return nil
}
