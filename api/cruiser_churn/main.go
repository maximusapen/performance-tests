
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	config "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/config"
	"github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/metrics"
	request "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/request"
	bluemix "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"
	"github.ibm.com/alchemy-containers/armada-performance/tools/crypto/utils"
)

// ChurnState defines an 'enum' for current state of cluster
type ChurnState int32

// States of cluster
const (
	ChurnNoCluster ChurnState = iota
	ChurnCreate
	ChurnUpdate
	ChurnDelete
	ChurnNoAction
	ChurnFreeze
)

var activeRequests int
var clusterPrefix string // Prefix to be used to Cruiser names
var existingClusters []ClusterChurnState
var openshiftChurn bool
var requestJobs chan request.Data
var startRequestNum int
var terminate bool               // Indicates if SIGTERM has been received
var totalWorkers int             // Total number of workers per user
var verbose bool                 // Output more detailed logging to stdout?
var maxClusterDeleteFailures = 3 // Maximum number of delete failures before givig up
var numThreads int               // Number of concurrent API requests
var frozenClusters int           // Number of clusters that didn't respond to multiple operations and so are locked out of future operations

// encyptionKey is the key used to decrypt sensitive data from the configuration file(s).
// It's value is baked in the executable at build time
var encryptionKey string

// ClusterChurnState  defines information tracking churn clusters
type ClusterChurnState struct {
	name              string
	masterKubeVersion string
	churnState        ChurnState
	deleteFailures    int
}

func getExistingClusters() []map[string]interface{} {
	requestData := request.Data{
		Action: config.ActionGetClusters}

	result := request.PerformRequest(requestData, true)

	if result.ActionFailed {
		fmt.Println("Request to find existing clusters failed")
		os.Exit(1)
	}

	var dat []map[string]interface{}
	if err := json.Unmarshal(result.Body, &dat); err != nil {
		panic(err)
	}

	return dat
}

func getClusterState(name string) (map[string]interface{}, request.Data, error) {
	var err error
	requestData := request.Data{
		Action:      config.ActionGetCluster,
		ClusterName: name}

	result := request.PerformRequest(requestData, true)

	if result.ActionFailed && verbose {
		fmt.Println("Request to get cluster state failed", result.ClusterName, result.StatusCode, result.Status)
	}

	var dat map[string]interface{}
	if err = json.Unmarshal(result.Body, &dat); err != nil {
		fmt.Println("ERROR: unmarshal of GetCluster response failed")
		dat = nil
	}

	return dat, result, err
}

func getClusterWorkers(name string) ([]interface{}, request.Data, error) {
	var err error
	requestData := request.Data{
		Action:      config.ActionGetClusterWorkers,
		ClusterName: name}

	result := request.PerformRequest(requestData, true)

	if result.ActionFailed && verbose {
		fmt.Println("Request to get cluster workers failed", result.ClusterName, result.StatusCode, result.Status)
	}

	var dat []interface{}
	if err = json.Unmarshal(result.Body, &dat); err != nil {
		fmt.Println("ERROR: unmarshal of GetClusterWorkers response failed")
		dat = nil
	}

	return dat, result, err
}

func getUpgradeVersion(currentVersion string, upgradeVersion string) string {
	currentVersions := strings.Split(currentVersion, ".")
	currentMajor, _ := strconv.Atoi(currentVersions[0])
	currentMinor, _ := strconv.Atoi(currentVersions[1])

	upgradeVersions := strings.Split(upgradeVersion, ".")
	upgradeMajor, _ := strconv.Atoi(upgradeVersions[0])
	upgradeMinor, _ := strconv.Atoi(upgradeVersions[1])

	if currentMajor != upgradeMajor {
		fmt.Println("Warning: Unlikely to upgrade due to change in major version", currentVersion, upgradeVersion)
	} else if currentMinor+1 < upgradeMinor {
		// Only upgrade one minor version at a time
		if openshiftChurn {
			return strconv.Itoa(upgradeMajor) + "." + strconv.Itoa(currentMinor+1) + "_openshift"
		}
		return strconv.Itoa(upgradeMajor) + "." + strconv.Itoa(currentMinor+1)
	}
	return upgradeVersion
}

func getKubeVersions() (string, string) {
	var major float64
	var minor float64

	requestData := request.Data{Action: config.ActionGetVersions}
	result := request.PerformRequest(requestData, true)

	if result.ActionFailed {
		fmt.Println("Request to find kube versions failed")
		os.Exit(1)
	}

	//var dat []map[string]interface{}
	var dat map[string][]map[string]interface{}
	if err := json.Unmarshal(result.Body, &dat); err != nil {
		panic(err)
	}

	var prop []map[string]interface{}
	if openshiftChurn {
		prop = dat["openshift"]
	} else {
		prop = dat["kubernetes"]
	}

	var kubeDefaultVersion string
	var kubeNextVersion string
	findNext := false
	for _, item := range prop {
		if item["default"].(bool) {
			major = item["major"].(float64)
			minor = item["minor"].(float64)
			// Don't include patch in version because patch level may become unavailable and creates would fail.
			kubeDefaultVersion = strconv.FormatFloat(major, 'f', 0, 64) + "." + strconv.FormatFloat(minor, 'f', 0, 64)
			if openshiftChurn {
				kubeDefaultVersion = kubeDefaultVersion + "_openshift"
			}
			findNext = true
		} else if findNext {
			nextMajor := item["major"].(float64)
			nextMinor := item["minor"].(float64)
			if nextMajor > major || (nextMajor == major && nextMinor > minor) {
				kubeNextVersion = strconv.FormatFloat(nextMajor, 'f', 0, 64) + "." + strconv.FormatFloat(nextMinor, 'f', 0, 64)
				if openshiftChurn {
					kubeNextVersion = kubeNextVersion + "_openshift"
				}
				break
			}
		}
	}

	return kubeDefaultVersion, kubeNextVersion
}

func createCluster(clusterIndex int) {
	clusterName := fmt.Sprintf("%s%d", clusterPrefix, startRequestNum)
	startRequestNum++
	if !terminate {
		existingClusters[clusterIndex].name = clusterName
		existingClusters[clusterIndex].churnState = ChurnCreate
		existingClusters[clusterIndex].masterKubeVersion = ""

		requestJobs <- request.Data{
			Action:            config.ActionCreateCluster,
			ClusterName:       clusterName,
			RequestNum:        clusterIndex,
			KubeUpdateVersion: "",
			TotalWorkers:      totalWorkers}
	} else {
		existingClusters[clusterIndex].churnState = ChurnNoCluster
	}

	activeRequests++
}

func main() {
	var totalClusters int                 // Total number of cruisers per user
	var zoneID string                     // The ID of the zone(datacenter) to process
	var machineType string                // Machine type, "free", "u2c.2x4", "b2c.4x16", "b2c.16x64", "b2c.32x128", "b2c.56x242", "u2c.2x4.encrypted", "b2c.4x16.encrypted", "b2c.16x64.encrypted", "b2c.32x128.encrypted", "b2c.56x242.encrypted"
	var upgradeKubeVersion string         // Upgrade Kubernetes version, major.minor.patch
	var defaultKubeVersion string         // Default Kubernetes version, major.minor.patch
	var testName string                   // Test name in Jenkins - only needed if sending alerts to RazeeDash
	var dbKey string                      // Metrics database key - only needed if sending metrics to database
	var debug bool                        // Output request and response summary to stdout?
	var sendMetrics bool                  // Send metrics data to Bluemix metric service?
	var workerPollInterval time.Duration  // Interval to poll for worker state/status changes
	var masterPollInterval time.Duration  // Interval to poll for master being ready
	var adminKubeConfig bool              // Use to retrieve admin certificate and PEM key for GetClusterConfig request
	var deleteResources bool              // Use to delete additional resources (e.g. storage) on cluster deletion request
	var showResources bool                // Use to retrieve additional resources (e.g. VLANs, subnets) for GetCluster request
	var privateVLAN, publicVLAN bool      // Flags to indicate VLAN for subnet creation action
	var action = config.ActionUnspecified // Mandatory flag to indicate API request to generate
	var monitor bool                      // Output monitoring data to stdout?
	var followKubeDefaultVersion bool     // Update defaultKubeVersion and upgradeKubeVersion when default kube version changes
	var carrierName string                // CarrierName to use when sending BOM version

	flag.StringVar(&clusterPrefix, "clusterNamePrefix", "perfCluster", "Name prefix of clusters. Use to specify name prefix for multiple patrols/cruisers")
	flag.IntVar(&numThreads, "numThreads", 0, "number of concurrent requests")
	flag.IntVar(&totalClusters, "clusters", 1, "total number of clusters per user")
	flag.IntVar(&totalWorkers, "workers", 1, "total number of worker nodes")
	flag.StringVar(&zoneID, "zoneId", "", "Identifier of zone (datacenter)")
	flag.StringVar(&machineType, "machineType", "", "machine type: free or u2c.2x4/b2c.4x16/b2c.16x64/b2c.32x128/b2c.56x242 [.encrypted]")
	flag.StringVar(&upgradeKubeVersion, "upgradeKubeVersion", "", "kubernetes version for upgrades: Run 'armada-perf-client -action=GetKubeVersions' for valid options. If not specified upgrades won't happen")
	flag.StringVar(&defaultKubeVersion, "defaultKubeVersion", "", "Default kubernetes version override: Run 'armada-perf-client -action=GetKubeVersions' for valid options")
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
	flag.DurationVar(&workerPollInterval, "workerPollInterval", (-1 * time.Second), "polling interval for checking cluster ready status; 0 means do not poll")
	flag.DurationVar(&masterPollInterval, "masterPollInterval", (-1 * time.Second), "polling interval for checking master ready status; 0 means do not poll")
	flag.BoolVar(&monitor, "monitor", false, "Output monitoring data on each request")
	flag.BoolVar(&followKubeDefaultVersion, "followKubeVersion", false, "Set/update default and upgrade kube version (upgradeKubeVersion) based on default kube version")
	flag.Var(&action, "action", "Armada API action. Valid options: "+strings.Join(config.Actions[1:].Strings(), ", "))
	flag.StringVar(&carrierName, "carrierName", "", "CarrierName to use when sending BOM version")

	flag.Parse()

	// Enforce mandatory action flag
	if action != config.ActionChurnClusters {
		fmt.Fprintf(os.Stderr, "Must specify -action ChurnClusters.\n")
		os.Exit(1)
	}

	if strings.Contains(defaultKubeVersion, "openshift") || strings.Contains(upgradeKubeVersion, "openshift") {
		openshiftChurn = true
	}

	basePath := config.GetConfigPath()
	var conf config.Config
	config.ParseConfig(filepath.Join(basePath, "perf.toml"), &conf)
	conf.Request.AdminConfig = adminKubeConfig
	conf.Request.PrivateVLAN = privateVLAN
	conf.Request.PublicVLAN = publicVLAN
	conf.Request.DeleteResources = deleteResources
	conf.Request.ShowResources = showResources
	// Churning real cruisers can use up a lot of subnets on our VLAN - so disable it for churn cruisers,
	// Except Openshift real cruisers, which need the subnet
	if totalWorkers > 0 && openshiftChurn {
		fmt.Println("Detected real openshift cruiser so will order subnet")
		conf.Softlayer.SoftlayerPortableSubnet = true
	} else {
		conf.Softlayer.SoftlayerPortableSubnet = false
	}

	// We want to use a special VLAN for churn to avoid filling up standard VLAN
	conf.Softlayer.SoftlayerPublicVLAN = conf.Softlayer.SoftlayerChurnPublicVLAN
	conf.Softlayer.SoftlayerPrivateVLAN = conf.Softlayer.SoftlayerChurnPrivateVLAN

	// Need to decrypt the api keys
	// Note setting the encryption key envvar here means it can be used by metrics code later on.
	os.Setenv(utils.KeyEnvVar, encryptionKey)
	if len(conf.Bluemix.APIKey) > 0 {
		var err error
		conf.Bluemix.APIKey, err = utils.Decrypt(conf.Bluemix.APIKey)
		if err != nil {
			log.Fatalf("Error decrypting IBM Cloud APIKey : %s\n", err.Error()) // pragma: allowlist secret
		}
	}
	totalRequests := totalClusters // Each user will make one request for each cluster

	if numThreads == 0 {
		fmt.Println("ERROR: numThreads needs to be set to something other than 0")
		os.Exit(1)
	}

	// Support SIGINT for clean exit with stats
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT)

	go func() {
		<-sigs
		terminate = true
		fmt.Println("Program will terminate after completion of current cluster CRUD")
		<-sigs
		panic("Exiting due to multiple SIGINT")
	}()

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

	if len(zoneID) == 0 {
		zoneID = conf.Location.Datacenter
	}

	rand.Seed(time.Now().UTC().UnixNano())

	requestJobs = make(chan request.Data, totalRequests)
	requestCompleted := make(chan request.Data, totalRequests)
	requestMetrics := make(map[config.ActionType]metrics.ArmadaMetrics)

	maxMetrics := 2 * numThreads
	requestMetrics[config.ActionCreateCluster] = make(metrics.ArmadaMetrics, maxMetrics)
	requestMetrics[config.ActionDeleteCluster] = make(metrics.ArmadaMetrics, maxMetrics)
	requestMetrics[config.ActionUpdateCluster] = make(metrics.ArmadaMetrics, maxMetrics)

	var createKubeVersion string

	if len(defaultKubeVersion) == 0 || followKubeDefaultVersion {
		request.InitRequests(machineType, defaultKubeVersion, "", &conf, verbose, debug, monitor, nil, nil, 0, nil, nil)
		defaultKube, nextKube := getKubeVersions()
		if followKubeDefaultVersion {
			upgradeKubeVersion = nextKube
			createKubeVersion = ""
		} else {
			createKubeVersion = defaultKube
		}
		defaultKubeVersion = defaultKube
	} else {
		createKubeVersion = defaultKubeVersion
	}
	request.InitRequests(machineType, createKubeVersion, "", &conf, verbose, debug, monitor, nil, nil, numThreads, &requestJobs, &requestCompleted)
	fmt.Println("Create kube version:", defaultKubeVersion, "upgrade kube version:", upgradeKubeVersion)

	startRequestNum = 1
	var existingClustersIndex int

	if action == config.ActionChurnClusters {
		dat := getExistingClusters()
		// See what is in carrier before deciding the action required.
		var maxIndex int
		existingClusters = make([]ClusterChurnState, totalClusters)
		for _, item := range dat {
			if strings.HasPrefix(item["name"].(string), clusterPrefix) {
				index, err := strconv.Atoi(strings.TrimPrefix(item["name"].(string), clusterPrefix))
				if err == nil {
					if existingClustersIndex >= totalClusters {
						fmt.Println("FATAL: The number of clusters to churn is less than the number of clusters in the carrier.")
						os.Exit(1)
					}
					existingClusters[existingClustersIndex] = ClusterChurnState{name: item["name"].(string),
						masterKubeVersion: item["masterKubeVersion"].(string), churnState: ChurnNoAction}
					existingClustersIndex++

					if index > maxIndex {
						maxIndex = index
					}
				}
			}
		}

		startRequestNum = maxIndex + 1
	}

	index := make(map[config.ActionType]int)
	var lastWriteTime time.Time

	pipelineMax := calculatePipelineMax()

churnClusters:
	for {
		// Keep the pipeline filled
		for activeRequests < pipelineMax && !terminate {

			var cluster ClusterChurnState
			var clusterName = ""
			var randAttempts int
			var clusterIndex int
			localTotalWorkers := totalWorkers
			localUpgradeKubeVersion := upgradeKubeVersion

		getClusterIndex:
			for {
				clusterIndex = int(rand.Int31n(int32(totalClusters)))
				cluster = existingClusters[clusterIndex]
				switch cluster.churnState {
				case ChurnNoCluster:
					action = config.ActionCreateCluster
					clusterName = fmt.Sprintf("%s%d", clusterPrefix, startRequestNum)
					cluster.name = clusterName
					cluster.churnState = ChurnCreate
					cluster.masterKubeVersion = ""
					startRequestNum++
					break getClusterIndex
				case ChurnNoAction:
					localTotalWorkers = 0
					clusterName = cluster.name

					if len(upgradeKubeVersion) == 0 || strings.HasPrefix(cluster.masterKubeVersion, strings.Trim(upgradeKubeVersion, "_openshift")) {
						action = config.ActionDeleteCluster
						cluster.churnState = ChurnDelete
					} else {
						action = config.ActionUpdateCluster
						cluster.churnState = ChurnUpdate
						localUpgradeKubeVersion = getUpgradeVersion(cluster.masterKubeVersion, upgradeKubeVersion)
					}
					break getClusterIndex
				default:
					// Handles ChurnCreate, ChurnUpdate, ChurnDelete, ChurnFreeze
					randAttempts++
					if randAttempts%1000 == 0 {
						// This should never happen but statistically it could and on May 7, 2019 the count ran up to 295040550 before process killed
						// Check if there is a slot/cruiser that can be acted upon. If not then there is a bug elsewhere in the code
						// so dump a bunch of data in hopes that with the rest of the logs the bug can be found.
						var foundCluster bool
						var maxedOutDeleteFailures int
						for _, cluster = range existingClusters {
							if cluster.churnState != cluster.churnState && cluster.churnState != ChurnNoAction {
								foundCluster = true
								break
							}
							if cluster.deleteFailures >= maxClusterDeleteFailures {
								maxedOutDeleteFailures++
							}
						}
						if !foundCluster {
							fmt.Printf("Random picking of cluster attempted: attempts %d, activeRequests %d, requests %d, complted %d, frozenClusters %d\n",
								randAttempts, activeRequests, len(requestJobs), len(requestCompleted), frozenClusters)
							fmt.Printf("ERROR: None of the %d slots/cruisers are available for an action.\n", len(existingClusters))
							fmt.Printf("       Suggests that activeRequests is out of line with real requests. Dumping existingClusters and exiting.\n")
							if maxedOutDeleteFailures > 1 {
								fmt.Println("WARNING: Found", maxedOutDeleteFailures, "clusters where", maxClusterDeleteFailures, "deletes failed. Requires manual delete.")
							}
						}
						for _, cluster = range existingClusters {
							fmt.Printf("%s %v %s\n", cluster.name, cluster.churnState, cluster.masterKubeVersion)
						}
						os.Exit(1)

					}
				}
			}
			existingClusters[clusterIndex] = cluster

			// Send the request to a pool of API request workers
			// At startup this can mean lots of requests hit the API concurrently, so slow them down
			time.Sleep(time.Second * 5)
			requestJobs <- request.Data{
				Action:            action,
				ClusterName:       clusterName,
				RequestNum:        clusterIndex,
				KubeUpdateVersion: localUpgradeKubeVersion,
				TotalWorkers:      localTotalWorkers}

			activeRequests++
		}

		var err error
		var data map[string]interface{}
		var response request.Data

		for depth := 1; depth > 0; depth = len(requestCompleted) {
			requestData := <-requestCompleted
			activeRequests--
			if sendMetrics {
				requestMetrics[requestData.Action][index[requestData.Action]] = requestData.Metrics
				requestMetrics[requestData.Action][index[requestData.Action]].ResponseTime = requestData.ResponseTime
				requestMetrics[requestData.Action][index[requestData.Action]].ActionTime = requestData.ActionTime
				requestMetrics[requestData.Action][index[requestData.Action]].ActionFailed = requestData.ActionFailed
				requestMetrics[requestData.Action][index[requestData.Action]].BackendFailed =
					requestData.ActionFailed && requestData.Failure != request.FailureUnspecified
				index[requestData.Action]++
			}

			initiateDelete := false
			cluster := existingClusters[requestData.RequestNum]
			switch cluster.churnState {
			case ChurnCreate:
				if requestData.StatusCode == http.StatusForbidden {
					// IBM Cloud Registry problem, try again
					createCluster(requestData.RequestNum)
				} else {
					// Check if cluster exists and in good state
					var receivedStatusInternalServerError bool
					for {
						data, response, err = getClusterState(cluster.name)
						if (err == nil && response.StatusCode != http.StatusInternalServerError) || response.StatusCode == http.StatusNotFound {
							break
						}
						// If StatusInternalServerError then suppress error messages. This will suppress any further work in this thread
						// until the server is working properly.
						if response.StatusCode == http.StatusInternalServerError {
							if !receivedStatusInternalServerError {
								fmt.Println("ERROR getCluster returned receivedStatusInternalServerError, waiting till getCluster is successful:",
									cluster.name, response.Action, response.Status)
							}
							receivedStatusInternalServerError = true
						} else {
							fmt.Println("ERROR getCluster:", cluster.name, response.Action, response.Status)
						}
						time.Sleep(10)
					}

					if response.StatusCode == http.StatusNotFound {
						// Cluster creation failed, queue a request for another cluster
						createCluster(requestData.RequestNum)
					} else if response.StatusCode/100 == 2 {
						if data["masterStatus"].(string) == "Ready" || data["masterStatus"].(string) == "VPN server configuration update in progress." || data["masterStatus"].(string) == "VPN server configuration update requested." {
							existingClusters[requestData.RequestNum].masterKubeVersion = data["masterKubeVersion"].(string)
							if followKubeDefaultVersion && !strings.HasPrefix(existingClusters[requestData.RequestNum].masterKubeVersion, strings.Trim(defaultKubeVersion, "_openshift")) {
								defaultKube, nextKube := getKubeVersions()
								// There may be hundreds of clusters with the old default kube version, so suppress future updates
								if defaultKube != defaultKubeVersion {
									defaultKubeVersion = defaultKube
									upgradeKubeVersion = nextKube
									request.SetKubeVersion(defaultKubeVersion)
									fmt.Println("Create kube version:", defaultKubeVersion, "upgrade kube version:", upgradeKubeVersion, " - update due to kube default version change")
								}
							}
							// Check if BOM version has changed, and update in InfluxDB if it has
							timestamp := time.Now()
							masterBOM := data["masterKubeVersion"].(string)
							fmt.Println("Created cluster with master BOM :", masterBOM)
							_, err = bluemix.WriteGrafanaBOMAnnotations(carrierName, masterBOM, bluemix.Master, timestamp)
							if err != nil {
								fmt.Println("ERROR writing master BOM version (", masterBOM, ") to InfluxDB: ", err)
							}
							if totalWorkers > 0 {
								// Also need to check Worker BOM
								workerData, _, err := getClusterWorkers(cluster.name)
								if err == nil {
									worker := workerData[0]
									workerDetails := worker.(map[string]interface{})

									workerBOM := workerDetails["kubeVersion"].(string)
									fmt.Println("Created cluster with worker BOM :", workerBOM)
									_, err = bluemix.WriteGrafanaBOMAnnotations(carrierName, workerBOM, bluemix.Worker, timestamp.Add(time.Minute*-10))
									if err != nil {
										fmt.Println("ERROR writing worker BOM version (", workerBOM, ") to InfluxDB: ", err)
									}
								} else {
									fmt.Println("ERROR getting worker BOM version", err)
								}
							}
						} else if requestData.ActionFailed {
							initiateDelete = true
						} else {
							fmt.Println("ERROR on create: masterStatus not 'Ready', or 'VPN server configuration update in progress.', or 'VPN server configuration update requested.' ", cluster.name, data["masterStatus"].(string))
							existingClusters[requestData.RequestNum].masterKubeVersion = defaultKubeVersion
						}
						existingClusters[requestData.RequestNum].churnState = ChurnNoAction
					} else {
						// Error on the side of assuming that cruiser was created to avoid the possibility of over populating
						// the carrier.
						// This problem has been solved for http.StatusInternalServerError with code above.
						fmt.Println("ERROR on create: Unexpected response from getStatus", cluster.name, response.Status)
						initiateDelete = true
					}
				}

			case ChurnUpdate:
				for {
					data, response, err = getClusterState(cluster.name)
					if err == nil || response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusConflict {
						break
					}
					fmt.Println("ERROR get cluster stage:", cluster.name, response.Action, response.Status)
					time.Sleep(10)
				}

				if response.StatusCode == http.StatusNotFound {
					fmt.Println("ERROR update: Couldn't find cluster after update:", cluster.name, response.Status)
					existingClusters[requestData.RequestNum] = ClusterChurnState{churnState: ChurnNoCluster}
					createCluster(requestData.RequestNum)
				} else if requestData.StatusCode == http.StatusConflict {
					// http.StatusConflict can be returned for multiple reasons. Start documenting so they can be handled:
					if strings.HasPrefix(data["masterKubeVersion"].(string), strings.Trim(upgradeKubeVersion, "_openshift")) {
						// 1) Cluster already upgraded to that version. Seen when cruiser_churn restarted and previous churn initiated an upgraded,
						//    and upgrade completed after cruiser_churn startup code grabbed list of all clusters.
						fmt.Println("INFO: Cluster already upgraded", cluster.name)
						existingClusters[requestData.RequestNum].masterKubeVersion = data["masterKubeVersion"].(string)
						existingClusters[requestData.RequestNum].churnState = ChurnNoAction
					} else {
						// 2) Reason behind the TODO below (i.e. unknown)
						//    TODO It would be nice to kick off another update, but the normal update request just returns StatsConflict
						//        Known fix is: `armada-data set Master -field UpdateState -value updated -pathvar MasterID=$MASTER`
						initiateDelete = true
					}
				} else {
					if response.StatusCode/100 == 2 {
						if requestData.ActionFailed {
							// This prevents a loop where GetCluster returns OK, but update hasn't happened, and thus kube version won't be changed.
							initiateDelete = true
						} else {
							existingClusters[requestData.RequestNum].masterKubeVersion = data["masterKubeVersion"].(string)
						}
					} else {
						fmt.Println("ERROR update: Unexpected error, mark as upgraded", cluster.name, requestData.Status, requestData.StatusCode)
						// Call could of failed for any number of reasons, most likely leaving a failed upgrade
						// Update the version to the requested version, in hopes that a subsequent delete will clear problem
						existingClusters[requestData.RequestNum].masterKubeVersion = upgradeKubeVersion
					}
					existingClusters[requestData.RequestNum].churnState = ChurnNoAction
				}

			case ChurnDelete:
				data, response, err = getClusterState(cluster.name)
				if response.StatusCode == http.StatusNotFound {
					existingClusters[requestData.RequestNum] = ClusterChurnState{churnState: ChurnNoCluster}

					// Ask for new cluster right away or randomness of approach could drop total clusters way below requested
					createCluster(requestData.RequestNum)
				} else {
					fmt.Println("ERROR delete: Unexpected error", cluster.name, response.Status)
					existingClusters[requestData.RequestNum].deleteFailures++
					if existingClusters[requestData.RequestNum].deleteFailures < maxClusterDeleteFailures {
						initiateDelete = true
					} else {
						// This could simply be the master status has a string that is unrecognized (see 'if master["masterStatus"] != nil' in request.go )
						fmt.Printf("REMEDIATE %d deletes failed, manual fix required to %s: %v\n", existingClusters[requestData.RequestNum].deleteFailures,
							cluster.name, response.Status)
						existingClusters[requestData.RequestNum].churnState = ChurnFreeze
						frozenClusters++
						pipelineMax = calculatePipelineMax()
					}
				}

			default:
				fmt.Printf("ERROR Unexpected churn state (%v) in response: %v\n", cluster.churnState, requestJobs)
			}

			if initiateDelete && !terminate && existingClusters[requestData.RequestNum].deleteFailures < maxClusterDeleteFailures {
				existingClusters[requestData.RequestNum].churnState = ChurnDelete
				requestJobs <- request.Data{
					Action:      config.ActionDeleteCluster,
					ClusterName: requestData.ClusterName,
					RequestNum:  requestData.RequestNum}

				activeRequests++
			}

			if monitor {
				if requestData.ActionFailed {
					fmt.Printf("%s: monitor %s %+v failed na %s\n", time.Now().Format(time.StampMilli), requestData.ClusterName, requestData.Action, requestData.ClusterID)
				} else {
					fmt.Printf("%s: monitor %s %+v succeeded %d %s\n", time.Now().Format(time.StampMilli), requestData.ClusterName, requestData.Action,
						requestData.ActionTime.Round(time.Minute)/time.Minute, requestData.ClusterID)
				}
			}

			// TODO move this outside loop, and change loop condition, so that if there isn't any completed CRUD and there are stats and it is >= 1min
			// then the metrics will be sent. Otherwise metrics that don't meet the < 60 conditional may not be sent for minutes.
			if sendMetrics {
				// If a buffer is full then wait until next window to write metrics
				if time.Since(lastWriteTime).Seconds() < 60 &&
					(index[config.ActionCreateCluster] == maxMetrics ||
						index[config.ActionUpdateCluster] == maxMetrics ||
						index[config.ActionDeleteCluster] == maxMetrics) {
					// Sleep for extra second due to rounding problems on the check below for >= 60
					time.Sleep(time.Second * time.Duration(61-time.Since(lastWriteTime).Seconds()))
				}
				// Metrics display better if workers set to 0
				metricsWorkers := totalWorkers
				if totalWorkers == -1 {
					metricsWorkers = 0
				}
				if time.Since(lastWriteTime).Seconds() >= 60 {
					if index[config.ActionCreateCluster] > 0 && (index[config.ActionCreateCluster] == maxMetrics || len(requestCompleted) == 0) {
						batchMetrics := requestMetrics[config.ActionCreateCluster][:index[config.ActionCreateCluster]]
						metrics.WriteArmadaMetrics(config.ActionCreateCluster, metricsWorkers, &batchMetrics, testName, dbKey)
						index[config.ActionCreateCluster] = 0
					}
					if index[config.ActionUpdateCluster] > 0 && (index[config.ActionUpdateCluster] == maxMetrics || len(requestCompleted) == 0) {
						batchMetrics := requestMetrics[config.ActionUpdateCluster][:index[config.ActionUpdateCluster]]
						metrics.WriteArmadaMetrics(config.ActionUpdateCluster, 0, &batchMetrics, testName, dbKey)
						index[config.ActionUpdateCluster] = 0
					}
					if index[config.ActionDeleteCluster] > 0 && (index[config.ActionDeleteCluster] == maxMetrics || len(requestCompleted) == 0) {
						batchMetrics := requestMetrics[config.ActionDeleteCluster][:index[config.ActionDeleteCluster]]
						metrics.WriteArmadaMetrics(config.ActionDeleteCluster, 0, &batchMetrics, testName, dbKey)
						index[config.ActionDeleteCluster] = 0
					}
					lastWriteTime = time.Now()
				}
			}
		}

		if terminate {
		emptyWorkQueue:
			for {
				select {
				case stopRequest := <-requestJobs:
					existingClusters[stopRequest.RequestNum].churnState = ChurnNoAction
				default:
					break emptyWorkQueue
				}
			}
			if len(requestCompleted) == 0 {
				remaining := 0
				for _, item := range existingClusters {
					if item.churnState == ChurnCreate || item.churnState == ChurnUpdate || item.churnState == ChurnDelete {
						remaining++
					}
				}

				fmt.Println("Cluster CRUD for", clusterPrefix, "still active", remaining)
				if remaining == 0 {
					break churnClusters
				}
			}
		} else if activeRequests >= 2*numThreads {
			for len(requestCompleted) == 0 && !terminate {
				time.Sleep(time.Second * 10)
			}
		}
	}
	// If requested, send metrics to Bluemix Metrics service
	if sendMetrics {
		lastWriteDuration := time.Since(lastWriteTime).Seconds()
		if lastWriteDuration < 60 {
			// Can't write metrics for a while so just sleep. Insures metrics writen ASAP
			time.Sleep(time.Second * time.Duration(60-lastWriteDuration))
		}
		// Metrics display better if workers set to 0
		metricsWorkers := totalWorkers
		if totalWorkers == -1 {
			metricsWorkers = 0
		}

		if index[config.ActionCreateCluster] > 0 {
			batchMetrics := requestMetrics[config.ActionCreateCluster][:index[config.ActionCreateCluster]]
			metrics.WriteArmadaMetrics(config.ActionCreateCluster, metricsWorkers, &batchMetrics, testName, dbKey)
			index[config.ActionCreateCluster] = 0
		} else if index[config.ActionUpdateCluster] > 0 {
			batchMetrics := requestMetrics[config.ActionUpdateCluster][:index[config.ActionUpdateCluster]]
			metrics.WriteArmadaMetrics(config.ActionUpdateCluster, 0, &batchMetrics, testName, dbKey)
			index[config.ActionUpdateCluster] = 0
		} else if index[config.ActionDeleteCluster] > 0 {
			batchMetrics := requestMetrics[config.ActionDeleteCluster][:index[config.ActionDeleteCluster]]
			metrics.WriteArmadaMetrics(config.ActionDeleteCluster, 0, &batchMetrics, testName, dbKey)
			index[config.ActionDeleteCluster] = 0
		}
	}

	fmt.Println("Done with cluster churn")
}

// Churning a small number of clusters, and clusters in ChurnFreeze state can impact number of clusters that can be churned.
func calculatePipelineMax() int {
	localPipelineMax := 2 * numThreads
	if len(existingClusters)-frozenClusters < localPipelineMax {
		localPipelineMax = len(existingClusters) - frozenClusters
	}
	if localPipelineMax <= 0 {
		fmt.Println("ERROR: pipelineMax <= 0, len(existingClusters):", len(existingClusters), ", frozenClusters:", frozenClusters)
		os.Exit(1)
	}
	return localPipelineMax
}
