
package cmdcluster

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-model/model"
	apiModelCommon "github.ibm.com/alchemy-containers/armada-model/model/api/json"
	apiModelV1 "github.ibm.com/alchemy-containers/armada-model/model/api/json/v1"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdworker"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/consts"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/metrics"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

// classicClusterCreateTP implements a worker thread pool for handling parallel classic cluster creation
func classicClusterCreateTP(c *cli.Context, cwg *sync.WaitGroup, requests <-chan apiModelV1.CreateClusterConfig, results chan<- clusterResponse) {
	endpoint := resources.GetArmadaEndpoint(c)

	// Block until cluster is created and report worker status at the specified interval ?
	pollIntervalStr, _ := cliutils.GetFlagString(c, models.PollIntervalFlagName, true, "Status polling interval", nil)
	timeoutStr, _ := cliutils.GetFlagString(c, models.TimeoutFlagName, true, "Status polling timeout", nil)
	workerCount, _ := cliutils.GetFlagInt(c, models.WorkersFlagName, false, "Worker Count", nil)

	pollInterval, err := time.ParseDuration(pollIntervalStr)
	if err != nil && len(pollIntervalStr) > 0 {
		fmt.Fprintf(c.App.Writer, "Invalid --%s %s - %s. Ignoring\n", models.PollIntervalFlagName, pollIntervalStr, err.Error())
	}

	for config := range requests {
		startTime := time.Now()
		if err := endpoint.CreateCluster(config); err != nil {
			switch val := err.(type) {
			case resources.NonCriticalError:
				fmt.Fprintf(c.App.Writer, "\nCluster creation request for classic cluster '%s' successful, but with warnings.", config.Name)
				fmt.Fprintf(c.App.Writer, "\n%s", val.Error())
			default:
				results <- clusterResponse{err: err}
				cwg.Done()
				continue
			}
		} else {
			fmt.Fprintf(c.App.Writer, "\nCluster creation request for classic cluster '%s' successful.\n", config.Name)
		}

		if pollInterval > 0 {
			if workerCount > 0 {
				// Poll on workers being ready
				c.Set(models.ClusterFlagName, config.Name) // Need to set the --cluster flag as required by "worker ls"
				if err := cmdworker.WorkerList(c); err != nil {
					results <- clusterResponse{err: err}
				} else {
					results <- clusterResponse{err: nil, duration: time.Since(startTime)}
				}
			} else {
				timeout, err := time.ParseDuration(timeoutStr)
				if err != nil && len(timeoutStr) > 0 {
					fmt.Fprintf(c.App.Writer, "Invalid --%s %s - %s. Ignoring\n", models.TimeoutFlagName, timeoutStr, err.Error())
				}

				// Poll on master being ready
				var masterReady, masterFailed bool

				fmt.Fprintln(c.App.Writer, "\n", startTime.Format(time.Stamp))
				fmt.Fprintf(c.App.Writer, "\tWaiting for '%s' Master to be Ready\n", config.Name)

				endpointV2 := resources.GetArmadaV2Endpoint(c)

				for !masterReady {
					if timeout > 0 && time.Since(startTime) > timeout {
						results <- clusterResponse{err: fmt.Errorf("Timeout waiting for classic cluster master(s) to be ready")}
						break
					}
					if _, v2Cluster, err := endpointV2.GetCluster(config.Name); err != nil {
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
			}
		} else {
			results <- clusterResponse{err: nil, duration: time.Since(startTime)}
		}
		cwg.Done()
	}
}

func classicClusterCreate(c *cli.Context) error {
	var cwg sync.WaitGroup

	infraConfig := cliutils.GetInfrastructureConfig().Classic

	// Initialize metrics data
	metricsData := cliutils.GetMetadataValue(c, models.MetricsFlagName).(*metrics.Data)
	metricsData.Command = fmt.Sprintf("%s-cluster", strings.Split(cliutils.GetCommandName(c), " ")[0])
	metricsData.Enabled = cliutils.GetFlagBool(c, models.MetricsFlagName)

	name, err := cliutils.GetFlagString(c, models.NameFlagName, false, "Name of the cluster to be created.", nil)
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

	kubeVersion, err := cliutils.GetFlagString(c, models.KubeVersionFlagName, true, "Cluster Kubernetes version", nil)
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

	podSubnet, err := cliutils.GetFlagString(c, models.PodSubnetFlagName, true, "", "")
	if err != nil {
		return err
	}

	serviceSubnet, err := cliutils.GetFlagString(c, models.ServiceSubnetFlagName, true, "", "")
	if err != nil {
		return err
	}

	dataCenter, err := cliutils.GetFlagString(c, models.ZoneFlagName, false, "IBM Cloud Zone", infraConfig.Zone)
	if err != nil {
		return err
	}

	noSubnet := cliutils.GetFlagBool(c, models.NoSubnetFlagName)

	machineType, err := cliutils.GetFlagString(c, models.MachineTypeFlagName, false, "Worker Node Machine Flavor", infraConfig.Flavor)
	if err != nil {
		return err
	}

	osName, err := cliutils.GetFlagString(c, models.OperatingSystemFlagName, true, "Operating System Name", nil)
	if err != nil {
		return err
	}

	disableEncryptionStr, err := cliutils.GetFlagString(c, models.DisableDiskEncryptionFlagName, true, "Disable Disk encryption", strconv.FormatBool(infraConfig.DisableDiskEncryption))
	if err != nil {
		return err
	}
	disableEncryption, err := strconv.ParseBool(disableEncryptionStr)
	if err != nil {
		return err
	}
	encrypted := !disableEncryption

	workerCount, err := cliutils.GetFlagInt(c, models.WorkersFlagName, true, "Number of worker nodes", consts.DefaultWorkerNum)
	if err != nil {
		return err
	}

	var hardware string
	isolationType, err := cliutils.GetFlagString(c, models.HardwareFlagName, false, "Worker Node Isolation (shared/dedicated)", infraConfig.Isolation)
	if err != nil {
		return err
	}
	switch strings.ToLower(isolationType) {
	case "": // do nothing
	case hardwareDedicated:
		hardware = models.IsolationPrivate
	case hardwareShared:
		hardware = models.IsolationPublic
	default:
		invalidValue := fmt.Sprintf(cliutils.InvalidValueMsg, models.HardwareFlagName, isolationType)
		permittedValues := fmt.Sprintf(cliutils.PermittedValuesMsg, models.HardwareFlagName, models.PermittedHardwareValues)

		return fmt.Errorf("%s\n%s", invalidValue, permittedValues)
	}

	privateServiceEndpointEnabled := cliutils.GetFlagBoolT(c, models.MasterPrivateServiceEndpointFlagName)
	publicServiceEndpointEnabled := cliutils.GetFlagBoolT(c, models.MasterPublicServiceEndpointFlagName)

	configTemplate := apiModelV1.CreateClusterConfig{
		ClusterConfig: apiModelV1.ClusterConfig{
			ClusterConfigCommon: apiModelCommon.ClusterConfigCommon{
				// Required
				Name: name,

				// Optional with default
				NoSubnet:      noSubnet,
				MasterVersion: kubeVersion,
				PodSubnet:     podSubnet,
				ServiceSubnet: serviceSubnet,
			},

			// Populate worker config
			WorkerConfig: apiModelV1.WorkerConfig{
				DataCenter:      dataCenter,
				PublicVlan:      vlanPublic,
				PrivateVlan:     vlanPrivate,
				MachineType:     machineType,
				OperatingSystem: osName,
				WorkerConfigCommon: apiModelCommon.WorkerConfigCommon{
					WorkerNum:      workerCount,
					Isolation:      hardware,
					DiskEncryption: encrypted,
				},
			},
			PrivateEndpointEnabled: privateServiceEndpointEnabled,
			PublicEndpointEnabled:  publicServiceEndpointEnabled,
		},
	}

	// Setup a pool of worker threads to handle cluster creation request(s)
	requests := make(chan apiModelV1.CreateClusterConfig, quantity)
	results := make(chan clusterResponse, quantity)
	cwg.Add(quantity)

	for t := 1; t <= threads; t++ {
		go classicClusterCreateTP(c, &cwg, requests, results)
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

	for cl := 1; cl <= quantity; cl++ {
		r := <-results
		if r.err != nil {
			return r.err
		}
		metricsData.C.Duration = append(metricsData.C.Duration, r.duration)
	}

	fmt.Fprintf(c.App.Writer, "Total Duration: %.1fs\n", endTime.Seconds())

	return nil
}
