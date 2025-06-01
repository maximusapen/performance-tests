
package cmdcluster

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-api-model/json/v2/requests"
	"github.ibm.com/alchemy-containers/armada-model/model"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cliutils"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/cmdworker"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/commands/consts"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/metrics"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/resources"
)

// vpcClusterCreateTP implements a worker thread pool for handling parallel vpc cluster creation
func vpcClusterCreateTP(c *cli.Context, cwg *sync.WaitGroup, requests <-chan requests.VPCCreateCluster, results chan<- clusterResponse) {
	endpoint := resources.GetArmadaV2Endpoint(c)

	// Block until cluster is created and report worker status at the specified interval ?
	pollIntervalStr, _ := cliutils.GetFlagString(c, models.PollIntervalFlagName, true, "Worker status polling interval", nil)
	timeoutStr, _ := cliutils.GetFlagString(c, models.TimeoutFlagName, true, "Status polling timeout", nil)
	workerCount, _ := cliutils.GetFlagInt(c, models.WorkersFlagName, false, "Worker Count", nil)

	pollInterval, err := time.ParseDuration(pollIntervalStr)
	if err != nil && len(pollIntervalStr) > 0 {
		fmt.Fprintf(c.App.Writer, "Invalid --%s %s - %s. Ignoring\n", models.PollIntervalFlagName, pollIntervalStr, err.Error())
	}

	for config := range requests {
		startTime := time.Now()
		if err := endpoint.CreateVPCCluster(config); err != nil {
			results <- clusterResponse{err: err}
		} else {
			fmt.Fprintf(c.App.Writer, "\nCluster creation request for %s cluster '%s' successful.\n", config.Provider, config.Name)

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

					for !masterReady {
						if timeout > 0 && time.Since(startTime) > timeout {
							results <- clusterResponse{err: fmt.Errorf("Timeout waiting for VPC cluster master(s) to be ready")}
							break
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
				}
			} else {
				results <- clusterResponse{err: nil, duration: time.Since(startTime)}
			}
		}
		cwg.Done()
	}
}

func vpcClassicClusterCreate(c *cli.Context) error {
	return vpcClusterCreate(c, models.ProviderVPCClassic, cliutils.GetInfrastructureConfig().VPCClassic)
}
func vpcNextGenClusterCreate(c *cli.Context) error {
	return vpcClusterCreate(c, models.ProviderVPCGen2, cliutils.GetInfrastructureConfig().VPC)
}

func vpcClusterCreate(c *cli.Context, provider string, infraConfig *cliutils.TomlVPCInfrastructureConfig) error {
	var err error
	var cwg sync.WaitGroup

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

	osName, err := cliutils.GetFlagString(c, models.OperatingSystemFlagName, true, "Operating System Name", nil)
	if err != nil {
		return err
	}

	// Comma separated list of zones (e.g. us-south-1,us-south-3 )
	zoneIDs, err := cliutils.GetFlagString(c, models.ZoneFlagName, false, "", infraConfig.Zone)
	if err != nil {
		return err
	}
	zones := strings.Split(zoneIDs, ",")

	vpcID, err := cliutils.GetFlagString(c, models.VPCIDFlagName, false, "Virtual Private Cloud Identifier", infraConfig.ID)
	if err != nil {
		return err
	}

	cosCRN, err := cliutils.GetFlagString(c, models.CosInstanceFlagName, true, "Cloud Object Storage Instance CRN", infraConfig.CosInstance)
	if err != nil {
		return err
	}

	flavor, err := cliutils.GetFlagString(c, models.MachineTypeFlagName, false, "Worker Node Machine Flavor", infraConfig.Flavor)
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

	disablePublicServiceEndpoint := false

	workerCount, err := cliutils.GetFlagInt(c, models.WorkersFlagName, true, "Number of worker nodes", consts.DefaultWorkerNum)
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

	// Build the zone configuration
	var zonesConfig []requests.VPCCreateClusterWorkerPoolZone
	for _, z := range zones {
		zc := requests.VPCCreateClusterWorkerPoolZone{
			CommonCreateClusterWorkerPoolZone: requests.CommonCreateClusterWorkerPoolZone{
				ID: z,
			},
			VPCCreateClusterWorkerPoolZoneFragment: requests.VPCCreateClusterWorkerPoolZoneFragment{
				SubnetID: infraConfig.Locations[z].SubnetID,
			},
		}

		if len(zc.SubnetID) == 0 {
			return fmt.Errorf("invalid zone specified: \"%s\"", z)
		}
		zonesConfig = append(zonesConfig, zc)
	}

	// Build the config object
	configTemplate := requests.VPCCreateCluster{
		CommonCreateCluster: requests.CommonCreateCluster{
			Name:                         name,
			KubeVersion:                  kubeVersion,
			PodSubnet:                    podSubnet,
			ServiceSubnet:                serviceSubnet,
			DisablePublicServiceEndpoint: disablePublicServiceEndpoint,
		},
		VPCCreateClusterFragment: requests.VPCCreateClusterFragment{
			Provider:       provider,
			COSInstanceCRN: cosCRN,
		},
		WorkerPool: requests.VPCCreateClusterWorkerPool{
			CommonCreateClusterWorkerPool: requests.CommonCreateClusterWorkerPool{
				Flavor:          flavor,
				DiskEncryption:  &encrypted,
				WorkerCount:     workerCount,
				OperatingSystem: osName,
			},
			VPCCreateClusterWorkerPoolFragment: requests.VPCCreateClusterWorkerPoolFragment{
				VPCID: vpcID,
			},
			Zones: zonesConfig,
		},
	}

	// Setup a pool of worker threads to handle cluster creation request(s)
	requests := make(chan requests.VPCCreateCluster, quantity)
	results := make(chan clusterResponse, quantity)
	cwg.Add(quantity)

	for t := 1; t <= threads; t++ {
		go vpcClusterCreateTP(c, &cwg, requests, results)
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
