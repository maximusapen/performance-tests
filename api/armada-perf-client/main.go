/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2020 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	bootstrap "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/bootstrap"
	carrier "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/carrier"
	config "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/config"
	deploy "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/deploy"
	"github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/metrics"
	request "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/request"
	"github.ibm.com/alchemy-containers/armada-performance/tools/crypto/utils"
)

// encyptionKey is the key used to decrypt sensitive data from the configuration file(s).
// It's value is baked in the executable at build time
var encryptionKey string

func main() {
	var singleClusterName string          // Name of a single cluster, used to target requests at a specific patrol/cruiser
	var clusterPrefix string              // Prefix to be used to Cruiser names
	var numThreads int                    // Number of concurrent API requests
	var totalRequestsPerUser int          // Total number of API requests per user
	var requestsPerAction int             // Total number of API requests per action
	var totalUsers int                    // Total unique users
	var totalClusters int                 // Total number of cruisers per user
	var totalWorkers int                  // Total number of workers per user
	var poolSize int                      // Total number of workers per zone
	var workerID string                   // The ID of the worker to process
	var workerPoolName string             // The ID of the worker pool to process
	var zoneID string                     // The ID of the zone(datacenter) to process
	var machineType string                // Machine type, "free", "u2c.2x4", "b2c.4x16", "b2c.16x64", "b2c.32x128", "b2c.56x242", "u2c.2x4.encrypted", "b2c.4x16.encrypted", "b2c.16x64.encrypted", "b2c.32x128.encrypted", "b2c.56x242.encrypted"
	var kubeVersion string                // Kubernetes version, major.minor.patch
	var testName string                   // Test name in Jenkins - only needed if sending alerts to RazeeDash
	var dbKey string                      // Metrics database key - only needed if sending metrics to database
	var debug bool                        // Output request and response summary to stdout?
	var verbose bool                      // Output more detailed logging to stdout?
	var sendMetrics bool                  // Send metrics data to Bluemix metric service?
	var workerPollInterval time.Duration  // Interval to poll for worker state/status changes
	var masterPollInterval time.Duration  // Interval to poll for master being ready
	var adminKubeConfig bool              // Use to retrieve admin certificate and PEM key for GetClusterConfig request
	var deleteResources bool              // Use to delete additional resources (e.g. storage) on cluster deletion request
	var showResources bool                // Use to retrieve additional resources (e.g. VLANs, subnets) for GetCluster request
	var privateVLAN, publicVLAN bool      // Flags to indicate VLAN for subnet creation action
	var useChurnVLAN bool                 // Use to determine whether to create clusters on the normal subnet or the churn subnet
	var action = config.ActionUnspecified // Mandatory flag to indicate API request to generate
	var monitor bool                      // Output monitoring data to stdout?

	flag.StringVar(&singleClusterName, "clusterName", "", "Name of cluster. Use to specify name of a single patrol/cruiser")
	flag.StringVar(&clusterPrefix, "clusterNamePrefix", "perfCluster", "Name prefix of clusters. Use to specify name prefix for multiple patrols/cruisers")
	flag.IntVar(&numThreads, "numThreads", 0, "number of concurrent requests")
	flag.IntVar(&totalUsers, "users", 1, "total number of users")
	flag.IntVar(&totalClusters, "clusters", 1, "total number of clusters per user")
	flag.IntVar(&totalWorkers, "workers", 1, "total number of worker nodes")
	flag.IntVar(&poolSize, "poolSize", 0, "total number of worker nodes in each zone")
	flag.IntVar(&requestsPerAction, "numRequests", 1, "total number of requests per action") // Used for primarily read requests
	flag.StringVar(&workerID, "workerId", "", "Identifier of worker. Used to specify the identity of a single worker node")
	flag.StringVar(&workerPoolName, "workerPoolName", "", "Identifier of worker pool. Used to specify the identity of a worker pool")
	flag.StringVar(&zoneID, "zoneId", "", "Identifier of zone (datacenter)")
	flag.StringVar(&machineType, "machineType", "", "machine type: free or u2c.2x4/b2c.4x16/b2c.16x64/b2c.32x128/b2c.56x242 [.encrypted]")
	flag.StringVar(&kubeVersion, "kubeVersion", "", "kubernetes version: Run 'armada-perf-client -action=GetKubeVersions' for valid options")
	flag.StringVar(&testName, "testname", "", "Test name in Jenkins - only needed if sending alerts to RazeeDash")
	flag.StringVar(&dbKey, "dbkey", "", "Metrics database key - only needed if sending metrics to database")
	flag.BoolVar(&debug, "debug", false, "Detailed logging output")
	flag.BoolVar(&verbose, "verbose", true, "Request and response summary logging output")
	flag.BoolVar(&sendMetrics, "metrics", false, "send metrics data to Bluemix metrics service")
	flag.BoolVar(&adminKubeConfig, "admin", false, "Retrieve Cluster Admin Configuration")
	flag.BoolVar(&deleteResources, "deleteResources", false, "Delete additional resources linked to the cluster")
	flag.BoolVar(&showResources, "showResources", false, "Show additional cluster resources")
	flag.BoolVar(&privateVLAN, "private", false, "Create Subnet on Private VLAN")
	flag.BoolVar(&publicVLAN, "public", false, "Create Subnet on Public VLAN")
	flag.BoolVar(&useChurnVLAN, "useChurnVLAN", false, "Create clusters on the churn VLAN")
	flag.DurationVar(&workerPollInterval, "workerPollInterval", (-1 * time.Second), "polling interval for checking cluster ready status; 0 means do not poll")
	flag.DurationVar(&masterPollInterval, "masterPollInterval", (-1 * time.Second), "polling interval for checking master ready status; 0 means do not poll")
	flag.BoolVar(&monitor, "monitor", false, "Output monitoring data on each request")
	flag.Var(&action, "action", "Armada API action. Valid options: "+strings.Join(config.Actions[1:].Strings(), ", "))

	flag.Parse()

	// Enforce mandatory action flag
	if action == config.ActionUnspecified {
		fmt.Fprintf(os.Stderr, "Must specify -action.\n")
		os.Exit(1)
	}

	if action == config.ActionCreateSubnet {
		if privateVLAN == publicVLAN {
			fmt.Fprintf(os.Stderr, "Specify either --private OR --public for the VLAN on which the subnet is to be created.\n")
			os.Exit(1)
		}
	}

	if (action == config.ActionCreateWorkerPool) || (action == config.ActionResizeWorkerPool) {
		if poolSize == 0 {
			fmt.Fprintf(os.Stderr, "Specify a valid ( >0 ) worker pool size.\n")
			os.Exit(1)
		}
	}
	// So, are you sitting comfortably ? Then I'll begin.
	// Once upon a time, Armada API supported specifying use of Bluemix organizations and so the code in here was written
	// and the author saw that it was good.
	// Then on the 5th day (actual number is probably > 5) the use of Bluemix accounts came forth, and the author looked at the code
	// in here and saw the it wasn't so good anymore.
	// And so the author commanded that until the point that someone could get around to changing the code to use accounts for different users
	// instead of organizations, then the support for multiple users would be no more.
	// p.s. the author isn't convinced that we actual really need this support, but just in case has left the framework within.
	if totalUsers > 1 {
		fmt.Fprintf(os.Stderr, "-users flag not currently supported.\n")
		os.Exit(1)
	}

	basePath := config.GetConfigPath()
	var conf config.Config
	var err error
	config.ParseConfig(filepath.Join(basePath, "perf.toml"), &conf)

	// Need to decrypt the api keys
	os.Setenv(utils.KeyEnvVar, encryptionKey)
	if len(conf.Bluemix.APIKey) > 0 {
		conf.Bluemix.APIKey, err = utils.Decrypt(conf.Bluemix.APIKey)
		if err != nil {
			log.Fatalf("Error decrypting IBM Cloud APIKey : %s\n", err.Error()) // pragma: allowlist secret
		}
	}
	if len(conf.Softlayer.SoftlayerAPIKey) > 0 {
		conf.Softlayer.SoftlayerAPIKey, err = utils.Decrypt(conf.Softlayer.SoftlayerAPIKey)
		if err != nil {
			log.Fatalf("Error decrypting Softlayer APIKey : %s\n", err.Error()) // pragma: allowlist secret
		}
	}

	conf.Request.AdminConfig = adminKubeConfig
	conf.Request.PrivateVLAN = privateVLAN
	conf.Request.PublicVLAN = publicVLAN
	conf.Request.DeleteResources = deleteResources
	conf.Request.ShowResources = showResources

	// If requested then use the Churn VLANs
	if useChurnVLAN {
		conf.Softlayer.SoftlayerPrivateVLAN = conf.Softlayer.SoftlayerChurnPrivateVLAN
		conf.Softlayer.SoftlayerPublicVLAN = conf.Softlayer.SoftlayerChurnPublicVLAN
	}

	startRequestNum := 1
	var existingClusters []string
	var existingClustersIndex int

	if action == config.ActionAlignClusters {
		// See what is in carrier before deciding the action required.
		requestJobs := make(chan request.Data, 1)
		requestCompleted := make(chan request.Data, 1)
		request.InitRequests(machineType, kubeVersion, workerPoolName, &conf, false, false, false, nil, nil, 1, &requestJobs, &requestCompleted)
		requestJobs <- request.Data{
			Action:         config.ActionGetClusters,
			ClusterName:    "",
			RequestNum:     1,
			WorkerID:       "",
			WorkerPoolName: "",
			ZoneID:         "",
			PoolSize:       0,
			TotalWorkers:   1}
		requestData := <-requestCompleted

		if requestData.ActionFailed {
			fmt.Println("Request to find existing clusters failed")
			os.Exit(1)
		}

		var dat []map[string]interface{}
		if err := json.Unmarshal(requestData.Body, &dat); err != nil {
			panic(err)
		}
		var maxIndex int
		existingClusters = make([]string, 0, 1500)
		for _, item := range dat {
			if strings.HasPrefix(item["name"].(string), clusterPrefix) {
				index, err := strconv.Atoi(strings.TrimPrefix(item["name"].(string), clusterPrefix))
				if err == nil {
					existingClusters = append(existingClusters, item["name"].(string))
					existingClustersIndex++

					if index > maxIndex {
						maxIndex = index
					}
				}
			}
		}

		totalClusters -= existingClustersIndex
		if totalClusters == 0 {
			fmt.Println("Request met by existing clusters in carrier")
			os.Exit(0)
		} else if totalClusters < 0 {
			totalClusters = 0 - totalClusters
			action = config.ActionDeleteCluster
			fmt.Println("Requesting", totalClusters, "be deleted")
		} else {
			startRequestNum = maxIndex + 1
			action = config.ActionCreateCluster
			fmt.Println("Requesting", totalClusters, "be created with starting index of", startRequestNum)
		}
	}

	if totalClusters > 1 {
		if action == config.ActionGetClusters {
			fmt.Fprintf(os.Stdout, "Ignoring -clusters for GetClusters action.\n")
			totalClusters = 1
		}
		if len(singleClusterName) > 0 {
			fmt.Fprintf(os.Stdout, "Ignoring -clusters. Specific cluster name specified.\n")
			totalClusters = 1
		}
	}

	totalRequestsPerUser = totalClusters // Each user will make one request for each cluster
	if !action.WorkerCreation() {
		totalWorkers = 0
	}
	if !action.HasCluster() {
		totalRequestsPerUser = 1 // Request specifies a specific cluster, so doesn't make sense (here at least) to be anything other than a single request for each user
		totalClusters = 0
	}

	totalRequests := totalRequestsPerUser * requestsPerAction
	if numThreads == 0 {
		// Default to processing ALL requests in parallel
		numThreads = totalUsers * totalRequests
	}

	// If not explicity specified, default machine type based on number of clusters/workers
	// (A free "patrol" account is limited to 1 cluster and worker).
	if len(machineType) == 0 {
		if action == config.ActionAddClusterWorkers || action == config.ActionCreateWorkerPool || totalWorkers > 1 || totalClusters > 1 {
			machineType = "u2c.2x4"
		} else {
			machineType = config.FreeAccountStr
		}
	}

	if masterPollInterval >= 0 {
		conf.Request.MasterPollInterval.Duration = masterPollInterval
	}

	if workerPollInterval >= 0 {
		conf.Request.WorkerPollInterval.Duration = workerPollInterval
	}

	// Populate etcd data that would be created during a carrier deployment
	if len(conf.Etcd.EtcdEndpoints) > 0 {
		carrier.InitCarrier(machineType, &conf)
	}

	// Initialize mock deploy and/or bootstrap if we're not running against a real deployment
	var mockDeploy *deploy.MockDeploy
	if conf.Deploy.DeployDummy {
		mockDeploy = deploy.InitMockDeploy(&conf, debug)
	}
	var mockBootstrap *bootstrap.MockBootstrap
	if conf.Bootstrap.BootstrapDummy {
		mockBootstrap = bootstrap.InitMockBootstrap(totalWorkers, &conf, debug)
	}

	if len(zoneID) == 0 {
		zoneID = conf.Location.Datacenter
	}

	requestJobs := make(chan request.Data, totalUsers*totalRequests)
	requestCompleted := make(chan request.Data, totalUsers*totalRequests)
	request.InitRequests(machineType, kubeVersion, workerPoolName, &conf, verbose, debug, monitor, mockDeploy, mockBootstrap, numThreads, &requestJobs, &requestCompleted)

	var requestMetrics = make(metrics.ArmadaMetrics, totalRequests, totalRequests)

	// For each user
	startTime := time.Now()
	for userID := 1; userID <= totalUsers; userID++ {
		// For each cluster
		for requestNum := 1; requestNum <= totalRequests; requestNum++ {
			// Generate a unique cluster name
			var clusterName = ""

			if action.HasCluster() {
				if len(singleClusterName) > 0 {
					clusterName = singleClusterName

				} else if existingClustersIndex > 0 && action == config.ActionDeleteCluster {
					clusterName = existingClusters[requestNum-1]
				} else {
					clusterName = fmt.Sprintf("%s%d", clusterPrefix, startRequestNum+requestNum-1)
				}
			}

			// Send the request to a pool of API request workers
			requestJobs <- request.Data{
				Action:         action,
				ClusterName:    clusterName,
				RequestNum:     requestNum,
				WorkerID:       workerID,
				WorkerPoolName: workerPoolName,
				ZoneID:         zoneID,
				PoolSize:       poolSize,
				TotalWorkers:   totalWorkers}

		}
	}

	var index int
	var lastWriteTime time.Time

	for requestIndex := 0; requestIndex < totalUsers*totalRequests; requestIndex++ {
		// Wait for all request threads to finish
		requestData := <-requestCompleted
		requestMetrics[index] = requestData.Metrics
		requestMetrics[index].ResponseTime = requestData.ResponseTime
		requestMetrics[index].ActionTime = requestData.ActionTime
		requestMetrics[index].ActionFailed = requestData.ActionFailed
		index++
		if monitor {
			if requestData.ActionFailed {
				fmt.Printf("%s: monitor %s %+v failed na %s\n", time.Now().Format(time.StampMilli), requestData.ClusterName, action, requestData.ClusterID)
			} else {
				fmt.Printf("%s: monitor %s %+v succeeded %d %s\n", time.Now().Format(time.StampMilli), requestData.ClusterName, action,
					requestData.ActionTime.Round(time.Minute)/time.Minute, requestData.ClusterID)
			}

			if sendMetrics && len(requestCompleted) == 0 {
				lastWriteDuration := time.Since(lastWriteTime).Seconds()
				if lastWriteDuration < 60 {
					// Can't write metrics for a while so just sleep. Insures metrics writen ASAP
					time.Sleep(time.Second * time.Duration(60-lastWriteDuration))
					if len(requestCompleted) > 0 {
						continue
					}
				}
				batchMetrics := requestMetrics[:index]
				metrics.WriteArmadaMetrics(action, totalWorkers, &batchMetrics, testName, dbKey)
				index = 0
				lastWriteTime = time.Now()
			}
		}
	}

	totalDuration := time.Since(startTime)
	throughput := float64(totalRequests) / totalDuration.Seconds()
	fmt.Printf("%s\tAction: %s, Throughput: %v req/sec, Total Duration: %.3fs\n", time.Now().Format(time.StampMilli), action.String(), int(throughput), totalDuration.Seconds())

	// If requested, send metrics to Bluemix Metrics service
	if sendMetrics {
		metrics.WriteArmadaMetrics(action, totalWorkers, &requestMetrics, testName, dbKey)
	}
}
