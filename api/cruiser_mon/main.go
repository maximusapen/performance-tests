/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	config "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/config"
	"github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/kube"
	"github.ibm.com/alchemy-containers/armada-performance/tools/crypto/utils"

	// Used only when turning on cluster debugging below
	//"github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/cluster"
	metricsservice "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"
	k8sv1 "k8s.io/api/core/v1"
)

// cruiserState ...
type cruiserState int

const (
	// Booting indicates config files found but cruiser kube couldn't be contacted
	Booting cruiserState = iota
	// Active the cruiser kube API request was successful at least once
	Active
	// BootFailure indicates that it took too long for cruiser kube to go from Booting to Active
	BootFailure
	// Deleted indicates that the configuration for the cruiser has been deleted
	Deleted
)

// Cruiser is created for each cruiser, matching the prefix, seen on the nfs mount
type Cruiser struct {
	name       string
	cube       kube.Kube
	state      cruiserState
	discovered time.Time
	replicas   int32
	namespace  string
}

// Response is retruned after trying to determine the availability of the cruiser
type Response struct {
	cruiser     Cruiser
	podRunning  bool
	portActive  bool
	kubeUsable  bool
	wasBooting  bool
	softFailure bool
	deleting    bool
	duration    time.Duration
	timeEnd     time.Time
}

var (
	carrier           kube.Kube
	carrierPods       map[string]k8sv1.Pod
	configFile        string
	configPath        string
	cruiserConfigPath string
	cruiserPrefix     string
	measurement       string
	cruisers          map[string]Cruiser
	debug             bool
	details           bool
	failedCruisers    map[string]time.Time
	loadConfigOnce    bool
	minLoop           time.Duration
	publishMetrics    bool
	requestCompleted  chan Response
	cruiserCrud       chan Cruiser
	signalCount       int
	timeout           time.Duration
	terminate         bool
	viableTimeout     time.Duration
	useConfigMaps     bool
	numThreads        int
	checkActive       bool
)

// encyptionKey is the key used to decrypt sensitive data from the configuration file(s).
// It's value is baked in the executable at build time
var encryptionKey string

// main ...
func main() {
	flag.StringVar(&cruiserPrefix, "prefix", "", "Prefix for the name of the cruisers")
	flag.StringVar(&cruiserConfigPath, "dir", "/mnt/nfs", "Directory where the cruiser configurations are stored")
	flag.StringVar(&configFile, "config", "", "Full path of configuration parameters file (supersedes -configpath)")
	flag.StringVar(&configPath, "configpath", "", "Path of directory where configuration files are located")
	flag.StringVar(&measurement, "measurement", "dummycruisermaster", "The name of the InfluxDB measurement to store results in")
	flag.BoolVar(&useConfigMaps, "configmaps", false, "Cruiser configuration is stored in config mapse")
	flag.BoolVar(&publishMetrics, "pub", true, "Publish metrics to the metric service")
	flag.BoolVar(&debug, "debug", false, "debug logging output")
	flag.BoolVar(&details, "details", false, "Output time of every service api request")
	flag.BoolVar(&checkActive, "active", true, "Check if the cruiser's port is active")
	flag.BoolVar(&loadConfigOnce, "once", false, "Only load config from nfs directory one time")
	flag.DurationVar(&timeout, "timeout", time.Duration(0*time.Minute), "Timeout for calls to kube")
	flag.DurationVar(&viableTimeout, "viable", time.Duration(30*time.Minute), "Duration after which cruiser is consider non viable")
	flag.DurationVar(&minLoop, "loop", time.Duration(5*time.Minute), "Minimum amount of time between iterations")
	flag.IntVar(&numThreads, "numthreads", 100, "Number of worker threads to use")
	flag.Parse()

	if debug {
		kube.Debug = true
		//cluster.Debug = true
	}

	var conf config.Config
	basePath := config.GetConfigPath()
	if len(configPath) > 0 {
		basePath = configPath
		if publishMetrics {
			metricsservice.SetConfigPath(basePath)
		}
	}
	if len(configFile) == 0 {
		configFile = filepath.Join(basePath, "perf.toml")
	}
	config.ParseConfig(configFile, &conf)

	// Need to decrypt the DB Password for metrics later
	// Note setting the encryption key envvar here means it can be used by metrics code later on.
	os.Setenv(utils.KeyEnvVar, encryptionKey)

	if !strings.HasPrefix(measurement, "dummycruisermaster") {
		fmt.Println("Warning: --measurement value does not begin with dummycruisermaster - this means the metrics code will not attempt to retry the connection if it fails, so results may not be sent to Influx.")
	}
	cruisers = make(map[string]Cruiser)
	failedCruisers = make(map[string]time.Time)

	if publishMetrics == false {
		fmt.Println("Warning: Metrics won't be published to metric service")
	}

	fmt.Printf("Kube call timeout is %v\n", timeout)

	fmt.Printf("Storing configs for %s under %s\n", cruiserPrefix, cruiserConfigPath)

	carrierKubeConfig := os.Getenv("KUBECONFIG")
	if len(carrierKubeConfig) == 0 {
		home := os.Getenv("HOME")
		carrierKubeConfig = home + "/.kube/config"
	}

	var err error
	carrier, err = kube.CreateKube(carrierKubeConfig, timeout)
	if err != nil {
		fmt.Println("Error calling CreateKube", err)
	}

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT)

	go func() {
		<-sigs
		terminate = true
		signalCount++
		if signalCount > 2 {
			panic("Exiting due to multiple SIGTIN/^c")
		}
		fmt.Println("Program will terminate at next interupt point")
		signal.Notify(sigs, syscall.SIGINT)
	}()

	firstIteration := true

	requestJobs := make(chan Cruiser, 2000)
	requestCompleted = make(chan Response, 2000)
	cruiserCrud = make(chan Cruiser, 1000)

	for j := 1; j <= numThreads; j++ {
		go worker(j, requestJobs)
	}

	if useConfigMaps {
		go monitorCruiserConfigMaps()
	} else {
		go monitorCruiserAdvanced()
	}
mainloop:
	for {
		if terminate {
			break mainloop
		}

		loopStart := time.Now()

		if !firstIteration {
			// Will check if pod is running before trying to ping it
			var tmpCarrierPods, err = getCarrierPods("kubx-masters")

			// Use the last list if there is an error getting list. That way if apiserver is down
			// then cruisers can still be tested
			if err == nil {
				carrierPods = tmpCarrierPods
			}

			min := time.Duration(time.Hour * 1000)
			max := time.Duration(0)
			sum := time.Duration(0)
			var count, bootingCount, failureCount, bootFailureCount, softFailureCount, deletingCount, portFailureCount int

			// Perform API performance test on active cruises
			for _, cruiser := range cruisers {
				requestJobs <- cruiser
			}

			cruiserCount := len(cruisers)

			for i := 0; i < cruiserCount; i++ {
				response := <-requestCompleted
				cruiser := response.cruiser
				cruisers[cruiser.name] = cruiser
				switch cruiser.state {
				case Active:
					if response.deleting {
						deletingCount++
						if _, ok := failedCruisers[cruiser.name]; ok {
							fmt.Printf("MTTR_RESULT: %s, %s Outage may be due to being deleted\n",
								response.timeEnd.Format("2006-01-02 15:04:05"), cruiser.name)
							delete(failedCruisers, cruiser.name)
						}
					} else if !response.wasBooting {
						if response.kubeUsable && response.duration > 0 {
							count++
							//fmt.Printf("%s %v\n", cruiser.name, duration)
							if min > response.duration {
								min = response.duration
							}
							if max < response.duration {
								max = response.duration
							}
							sum = sum + response.duration

							if response.duration > time.Duration(time.Second) {
								fmt.Printf("WARNING: %s High duration of %v for %s\n", response.timeEnd.Format("2006-01-02 15:04:05"), response.duration, cruiser.name)
							}

							if _, ok := failedCruisers[cruiser.name]; ok {
								statOutage := failedCruisers[cruiser.name]
								fmt.Printf("MTTR_DATA: %s, %s, true, true, true\n", response.timeEnd.Format("2006-01-02 15:04:05"), cruiser.name)
								fmt.Printf("MTTR_RESULT: %s, %s, %s, %v\n", response.timeEnd.Format("2006-01-02 15:04:05"),
									cruiser.name, statOutage.Format("2006-01-02 15:04:05"), response.timeEnd.Sub(statOutage))
								delete(failedCruisers, cruiser.name)
							}

							if response.softFailure {
								softFailureCount++
							}
						}
						if response.podRunning == false || response.portActive == false || response.kubeUsable == false {
							if response.portActive == false {
								portFailureCount++
							} else {
								failureCount++
							}
							fmt.Printf("MTTR_DATA: %s, %s, %t, %t, %t %t\n", response.timeEnd.Format("2006-01-02 15:04:05"),
								cruiser.name, response.podRunning, response.portActive, response.kubeUsable, response.softFailure)
							if _, ok := failedCruisers[cruiser.name]; ok == false {
								failedCruisers[cruiser.name] = time.Now()
							}
						} else if details {
							fmt.Printf("%s %v\n", cruiser.name, response.duration)
						}
					} else {
						bootingCount++
					}
				case Booting:
					bootingCount++
				case BootFailure:
					bootFailureCount++
				}
			}

			// TODO time report is inconsistent since it depends on how long it took to collect the stats. Easy to solve for
			// the following Printfs, not so much for reporting to the metric service which is more important.
			// Report metrics
			if count > 0 {
				fmt.Printf("%s, %d, %d, %d, %d, %d, %d, %d, %v, %v, %v\n", time.Now().Format("2006-01-02 15:04:05"),
					count, portFailureCount, failureCount, softFailureCount, bootingCount, bootFailureCount, deletingCount, min, max, sum/time.Duration(count))
			} else {
				fmt.Printf("%s, %d, %d, %d, %d, %d, %d, %d, n/a, n/a, n/a\n", time.Now().Format("2006-01-02 15:04:05"),
					count, portFailureCount, failureCount, softFailureCount, bootingCount, bootFailureCount, deletingCount)
			}
			if publishMetrics {

				// New metrics processing will handle adding carrier name as part of metrics processing - so don't need carrier name here
				metricsPrefix := strings.Join([]string{measurement}, ".")

				// Lets fix the metrics we want here for now.
				// For future, would be nice to define these in a configuration file
				var bm []metricsservice.BluemixMetric
				if count > 0 {
					bm = make([]metricsservice.BluemixMetric, 10)
					bm[7] = metricsservice.BluemixMetric{
						Name:  metricsPrefix + ".kube_api.services.Latency_Min.min",
						Value: min.Seconds() * 1e3,
					}
					bm[8] = metricsservice.BluemixMetric{
						Name:  metricsPrefix + ".kube_api.services.Latency_Max.max",
						Value: max.Seconds() * 1e3,
					}
					bm[9] = metricsservice.BluemixMetric{
						Name:  metricsPrefix + ".kube_api.services.Latency_Mean.sparse-avg",
						Value: (sum / time.Duration(count)).Seconds() * 1e3,
					}
				} else {
					bm = make([]metricsservice.BluemixMetric, 7)
				}

				bm[0] = metricsservice.BluemixMetric{
					Name:  metricsPrefix + ".kube_api.services.Success_Count.max",
					Value: count,
				}
				bm[1] = metricsservice.BluemixMetric{
					Name:  metricsPrefix + ".kube_api.services.Failure_Count.max",
					Value: failureCount,
				}
				bm[2] = metricsservice.BluemixMetric{
					Name:  metricsPrefix + ".Boot_Failure_Count.max",
					Value: bootFailureCount,
				}
				bm[3] = metricsservice.BluemixMetric{
					Name:  metricsPrefix + ".Booting_Count.max",
					Value: bootingCount,
				}
				bm[4] = metricsservice.BluemixMetric{
					Name:  metricsPrefix + ".kube_api.services.Soft_Failure_Count.max",
					Value: softFailureCount,
				}
				bm[5] = metricsservice.BluemixMetric{
					Name:  metricsPrefix + ".Deleting.max",
					Value: deletingCount,
				}
				bm[6] = metricsservice.BluemixMetric{
					Name:  metricsPrefix + ".port.Failure_Count.max",
					Value: portFailureCount,
				}

				metricsservice.WriteBluemixMetrics(bm, true, measurement, "")
			}
		}

		waitForLoopDuration("Response", loopStart)

		// Carrier CRUD loop
	crudloop:
		for {
			select {
			case cruiser, ok := <-cruiserCrud:
				if ok {
					if debug {
						fmt.Printf("CRUD loop received %s - %v\n", cruiser.name, cruiser.state)
					}
					switch cruiser.state {
					case Deleted:
						delete(cruisers, cruiser.name)
						dirToDelete := cruiserConfigPath + "/" + cruiser.name
						err := os.RemoveAll(dirToDelete)
						if err != nil {
							fmt.Printf("ERROR: Deleting directory - will ignore %s: %v\n", dirToDelete, err)
						}
					case Booting:
						cruisers[cruiser.name] = Cruiser{name: cruiser.name, state: Booting, cube: cruiser.cube,
							discovered: cruiser.discovered, replicas: cruiser.replicas, namespace: cruiser.namespace}
					}
				} else {
					fmt.Println("CRUD channel was closed")
					break crudloop
				}
			default:
				if debug {
					fmt.Println("No CRUD, exiting loop")
				}
				break crudloop
			}
		}

		firstIteration = false
	}

	for cruiserName, statOutage := range failedCruisers {
		fmt.Printf("MTTR_OUTSTANDING: %s, %s, %s, %v\n", time.Now().Format("2006-01-02 15:04:05"),
			cruiserName, statOutage.Format("2006-01-02 15:04:05"), time.Since(statOutage))
	}

	close(requestJobs)
	close(requestCompleted)
	close(cruiserCrud)
}

// waitForLoopDuration sleeps until the next action is required
func waitForLoopDuration(loopName string, loopStart time.Time) {
	// Wait till next cycle is due.
	loopDuration := time.Since(loopStart)
	if minLoop > loopDuration {
		if debug {
			fmt.Printf("%s loop sleep for %v\n", loopName, minLoop-loopDuration)
		}
		sleep := time.Second * 10
		for minLoop > loopDuration {
			if minLoop-loopDuration < sleep {
				sleep = minLoop - loopDuration
			}
			time.Sleep(sleep)
			loopDuration = loopDuration + sleep
			if terminate {
				return
			}
		}
	}
}

// getCarrierPods retrieves a list of pods for the purpose of filtering for cruiser state
func getCarrierPods(namespace string) (map[string]k8sv1.Pod, error) {
	podPrefix := "master-" + cruiserPrefix
	podList := make(map[string]k8sv1.Pod)
	pods, err := carrier.GetPodStatus(namespace)
	if err != nil {
		fmt.Println("Couldn't get a list of cruisers via carrier kube")
	} else {
		for _, pod := range pods.Items {
			if strings.HasPrefix(pod.Name, podPrefix) {
				dashCnt := strings.Count(pod.Name, "-")

				splitName := strings.Split(pod.Name, "-")
				podName := splitName[1]
				for i := 2; i <= dashCnt-2; i++ {
					podName = podName + "-" + splitName[i]
				}
				if _, ok := podList[podName]; ok {
					// Want list to say cruiser is running if at least one master is running
					if string(podList[podName].Status.Phase) != "Running" {
						podList[podName] = pod
					} else if debug {
						fmt.Printf("%s pod would overwrite Running with %s\n", podName, string(podList[podName].Status.Phase))
					}
				} else {
					podList[podName] = pod
				}
			}
		}
	}
	return podList, err
}

// worker provides a pool of workers to determine cruiser state
func worker(id int, jobs <-chan Cruiser) {
workerloop:
	for cruiser := range jobs {
		if terminate {
			break workerloop
		}

		if debug {
			fmt.Printf("%s\t result: %d - %s %v\n", time.Now().Format(time.StampMilli), id, cruiser.name, cruiser.state)
		}
		switch cruiser.state {
		case Active:
			requestCompleted <- handleActiveCruiser(cruiser)
		case Booting:
			requestCompleted <- handleBootingCruiser(cruiser)
		case BootFailure:
			requestCompleted <- Response{cruiser: cruiser}
		}
	}
}

// handleActiveCruiser polls to determine state of cruiser
func handleActiveCruiser(cruiser Cruiser) Response {
	var response Response
	response.cruiser = cruiser

	// Filtering on pod status will help reduced wasted polling. Given the large
	// numbers of cruisers being polled it is possible that the status has changed
	// since it was gathered. Its worth may be limited.
	if cruiser.namespace == "kubx-masters" {
		response.podRunning = string(carrierPods[cruiser.name].Status.Phase) == "Running"
	} else {
		// OpenShift 4 runs in 42 pods, most of then in HA configuration, running in its own namespace.
		// It doesn't make sense to make a separate "get pods" request per cluster, or to get a list of
		// all clusters in all namespaces. Nor does it make sense to check for "Running" all pods for the cluster.
		// So just punt on podRunning test for OpenShift 4 clusters
		response.podRunning = true
	}

	if response.podRunning {
		if checkActive {
			active, err := cruiser.cube.PortActive()
			response.portActive = active
			if err != nil {
				softMessage := ""
				dialTime := time.Now()
				if cruiser.replicas > 1 {
					active, err2 := cruiser.cube.PortActive()
					response.portActive = active
					if err2 == nil {
						response.softFailure = true
						response.portActive = true
						softMessage = "softFailure - "
					}
				}
				response.timeEnd = time.Now()
				if response.portActive == false {
					if !hasDeploymentReplicas(cruiser) {
						cruiserCrud <- Cruiser{name: cruiser.name, state: Deleted, namespace: cruiser.namespace}
						if debug {
							fmt.Printf("Removing because master deployment doesn't exist: %s\n", cruiser.name)
						}
						response.deleting = true
					} else {
						fmt.Printf("ERROR: %s Port dial failed for %s - %s%v\n", response.timeEnd.Format("2006-01-02 15:04:05"), cruiser.name, softMessage, err)
					}
					return response
				} else if cruiser.replicas > 1 {
					fmt.Printf("ERROR: %s 1st Port dial failed for %s - %v\n", dialTime.Format("2006-01-02 15:04:05"), cruiser.name, err)
				}
			}
		} else {
			// TODO should have "unknown" state. For now just set true so monitoring loop doesn't mark it as a failure
			response.portActive = true
		}
		timeStart := time.Now()
		_, err := cruiser.cube.GetServices()
		response.timeEnd = time.Now()
		response.duration = response.timeEnd.Sub(timeStart)

		if err != nil {
			softMessage := ""

			if !hasDeploymentReplicas(cruiser) {
				cruiserCrud <- Cruiser{name: cruiser.name, state: Deleted, namespace: cruiser.namespace}
				if debug {
					fmt.Printf("Removing because master deployment doesn't exist: %s\n", cruiser.name)
				}
				response.deleting = true
			} else if cruiser.replicas > 1 {
				fmt.Println("ERROR: 1st Error getting services: ", cruiser.name, response.duration, err)
				cruiser.cube.ResetConnections(timeout)
				timeStart := time.Now()
				_, err2 := cruiser.cube.GetServices()
				response.timeEnd = time.Now()
				response.duration = response.timeEnd.Sub(timeStart)
				if err2 == nil {
					response.softFailure = true
					response.kubeUsable = true
					softMessage = " softFailure"
				}
			}
			fmt.Printf("ERROR: Error getting services: %s %d %s%s %s\n", cruiser.name, cruiser.replicas, response.duration, softMessage, err)
		} else {
			response.kubeUsable = true
		}
	} else {
		// There aren't any masters in "Running" state. This could be because the cruiser is booting, or because
		// there are problems with the cruiser or because the cruiser is being deleted. Since code doesn't "see" cruiser
		// until GetDeploymentReplicas() returns successfully then a failure from GetDeploymentReplicas() should indicate
		// that the cruiser has been deleted.
		if !hasDeploymentReplicas(cruiser) {
			cruiserCrud <- Cruiser{name: cruiser.name, state: Deleted, namespace: cruiser.namespace}
			if debug {
				fmt.Printf("Removing because master deployment doesn't exist: %s\n", cruiser.name)
			}
			response.deleting = true
		}
	}
	return response
}

// hasDeploymentReplicas ...
func hasDeploymentReplicas(cruiser Cruiser) bool {
	var deploymentName string
	if cruiser.namespace == "kubx-masters" {
		deploymentName = "master-" + cruiser.name
	} else {
		// OpenShift 4 clusters have many deployments, almost all with multiple pods. Key off of
		// just the apiserver since that is what the program will communicate with.
		deploymentName = "kube-apiserver"
	}

	_, replicaErr := carrier.GetDeploymentReplicas(cruiser.namespace, deploymentName)
	if replicaErr != nil && strings.Contains(replicaErr.Error(), "\""+deploymentName+"\" not found") {
		return false
	}
	return true
}

// handleBootingCruiser move booting cruisers to active or boot failure
func handleBootingCruiser(cruiser Cruiser) Response {
	var response Response

	// Check that kube port is accepting connections
	var isReachable bool
	active, _ := cruiser.cube.PortActive()
	if active {
		_, err := cruiser.cube.GetServices()
		if err == nil {
			isReachable = true
			// Don't record the first successful requests since they reflect the costs
			// of initializing the interface
			if debug {
				fmt.Printf("Active %s\n", cruiser.name)
			}
			return Response{cruiser: Cruiser{name: cruiser.name, state: Active, cube: cruiser.cube,
				discovered: cruiser.discovered, replicas: cruiser.replicas, namespace: cruiser.namespace}, wasBooting: true}
		}
	}

	if isReachable == false && time.Since(cruiser.discovered) > viableTimeout {
		fmt.Printf("Boot failed %s, timed out at %v\n", cruiser.name, time.Since(cruiser.discovered))
		return Response{cruiser: Cruiser{name: cruiser.name, state: BootFailure, cube: cruiser.cube,
			discovered: cruiser.discovered, replicas: cruiser.replicas, namespace: cruiser.namespace}}
	}
	response.cruiser = cruiser
	return response
}

// monitorCruiserConfigMaps finds new and deleted cruiser configs
func monitorCruiserConfigMaps() {
	firstIteration := true
	configMapPrefix := "master-" + cruiserPrefix

	// Tracks outstanding cruisers. There is the risk that this will get of sync with cruiser[] but
	// it is necissary since this method runs on a separate thread
	foundCruisers := make(map[string]bool)

	// Create directory for storing config files
	if _, err := os.Stat(cruiserConfigPath); os.IsNotExist(err) {
		err := os.MkdirAll(cruiserConfigPath, os.ModePerm)
		if err != nil {
			fmt.Printf("ERROR: Creating directory %s: %v\n", cruiserConfigPath, err)
			os.Exit(1)
		}
	}

	// Find new configurations and remove cruisers where the configuration has been removed
configloop:
	for {
		if terminate {
			break configloop
		}
		if debug {
			fmt.Printf("Monitor config maps loop\n")
		}

		loopStart := time.Now()

		if loadConfigOnce == false || firstIteration {
			foundCnt := 0

			for name := range foundCruisers {
				foundCruisers[name] = false
			}

			// Get master-${cruiser}-config and master-${cruiser}-cert config maps
			timeStart := time.Now()
			configMaps, err := carrier.GetConfigMapsByNamespace("kubx-masters")
			timeEnd := time.Now()
			cmDuration := timeEnd.Sub(timeStart)

			metricsPrefix := strings.Join([]string{measurement}, ".")
			bm := metricsservice.BluemixMetric{
				Name:  metricsPrefix + ".configmap_list.Latency.max",
				Value: cmDuration,
			}
			metricsservice.WriteBluemixMetrics([]metricsservice.BluemixMetric{bm}, true, measurement, "")

			if err != nil {
				fmt.Println("ERROR: Listing config maps", err)
			} else {
				for _, configMap := range configMaps.Items {
					//fmt.Println("Looking at config map ", configMap.Name)
					if strings.Contains(configMap.Name, configMapPrefix) && strings.HasSuffix(configMap.Name, "-config") {
						deployName := strings.TrimSuffix(configMap.Name, "-config")
						cruiserName := strings.TrimPrefix(deployName, "master-")

						if _, ok := foundCruisers[cruiserName]; ok == false {
							if debug {
								fmt.Println("Using config map: ", configMap.Name)
							}
							//fmt.Printf("Found %s\n", cruiserName)

							// Store cruiser kubernetes configuration locally
							configBase := cruiserConfigPath + "/" + cruiserName
							if _, err := os.Stat(configBase); os.IsNotExist(err) {
								err := os.MkdirAll(configBase, os.ModePerm)
								if err != nil {
									// Bad situation which will likely continue to happen, but never should
									fmt.Printf("ERROR: Creating directory %s: %v\n", configBase, err)
									continue
								}
							}

							config := configBase + "/admin-kubeconfig"
							storeConfigMapToFile(configMap.Data, "admin-kubeconfig", config)
							if err != nil {
								fmt.Println("Error: Failed to setup kubeconfig", err)
							}

							err := setupKube("kubx-masters", cruiserName, config, "master-"+cruiserName)
							if err == nil {
								foundCruisers[cruiserName] = true
							}
						} else {
							foundCnt++
							foundCruisers[cruiserName] = true
						}
					}
				}

				if foundCnt < len(foundCruisers) {
					for name := range foundCruisers {
						if foundCruisers[name] == false {
							delete(foundCruisers, name)
							cruiserCrud <- Cruiser{name: name, state: Deleted}
							if debug {
								fmt.Printf("Removing deleted configuration: %s\n", name)
							}
						}
					}
				}
			}
		}
		waitForLoopDuration("Cruiser config", loopStart)
		firstIteration = false
	}
}

// monitorCruiserAdvanced finds new and deleted cruiser configs
func monitorCruiserAdvanced() {
	firstIteration := true
	namespaceRegex, err := regexp.Compile("master-[a-z0-9]{20}")
	if err != nil {
		fmt.Println("ERROR: Couldn't compile namespace regex")
		os.Exit(1)
	}

	secretRegex, err := regexp.Compile("[a-z0-9]{20}-secrets")
	if err != nil {
		fmt.Println("ERROR: Couldn't compile secret regex")
		os.Exit(1)
	}

	hypershiftSecretRegex, err := regexp.Compile("[a-z0-9-]{10}-admin-kubeconfig")
	if err != nil {
		fmt.Println("ERROR: Couldn't compile secret regex")
		os.Exit(1)
	}

	// Tracks outstanding cruisers. There is the risk that this will get of sync with cruiser[] but
	// it is necissary since this method runs on a separate thread
	foundCruisers := make(map[string]bool)

	// Create directory for storing config files
	if _, err := os.Stat(cruiserConfigPath); os.IsNotExist(err) {
		err := os.MkdirAll(cruiserConfigPath, os.ModePerm)
		if err != nil {
			fmt.Printf("ERROR: Creating directory %s: %v\n", cruiserConfigPath, err)
			os.Exit(1)
		}
	}

	// Two methods are used to find out what clusters exist on a carrier
	// 1) Get a list of -n kubx-masters secrets named <cluster id>-secrets - classic and OpenShift 3 clusters
	//    Get the cluster's '*-config' and '*-certs' config maps to create the kubeconfig
	//    This method is used because the size, and thus transfer volume, of cluster config maps can be quite large.
	// 2) Get a list of namespaces namaed master-<cluster id> - OpenShift 4 clusters
	//    Get the cluster's 'pki.data' secret which contains the kubeconfig under 'pki'
	// If these elements are removed then it is assumed that the cluster has been deleted
	// It is very possible that though a cluster has been identified, the components needed
	// to construct the kubeconfig aren't yet available. Deal with it.
configloop:
	for {
		if terminate {
			break configloop
		}
		if debug {
			fmt.Printf("Monitor config maps loop\n")
		}

		loopStart := time.Now()

		if loadConfigOnce == false || firstIteration {
			foundCnt := 0
			var listError bool

			for name := range foundCruisers {
				foundCruisers[name] = false
			}

			// Get master-${cruiser}-config and master-${cruiser}-cert config maps
			timeStart := time.Now()
			secrets, err := carrier.GetSecretsByNamespace("kubx-masters")
			timeEnd := time.Now()
			cmDuration := timeEnd.Sub(timeStart)

			metricsPrefix := strings.Join([]string{measurement}, ".")
			bm := metricsservice.BluemixMetric{
				Name:  metricsPrefix + ".secret_list.Latency.max",
				Value: cmDuration,
			}
			metricsservice.WriteBluemixMetrics([]metricsservice.BluemixMetric{bm}, true, measurement, "")

			if err != nil {
				fmt.Println("ERROR: Listing secrets", err)
				listError = true
			} else {
				for _, secret := range secrets.Items { // pragma: allowlist secret
					// fmt.Println("Looking at secret", secret.Name)
					if secretRegex.MatchString(secret.Name) {
						cruiserName := strings.TrimSuffix(secret.Name, "-secrets")

						if _, ok := foundCruisers[cruiserName]; ok == false {
							foundCruisers[cruiserName] = extractKubeConfig("kubx-masters", cruiserName)
						} else {
							foundCnt++
							foundCruisers[cruiserName] = true
						}
					}

				}

				// Get master-${cruiser} namespaces
				timeStart := time.Now()
				namespaces, err := carrier.GetNamespaces()
				timeEnd := time.Now()
				cmDuration := timeEnd.Sub(timeStart)

				metricsPrefix := strings.Join([]string{measurement}, ".")
				bm := metricsservice.BluemixMetric{
					Name:  metricsPrefix + ".namespace_list.Latency.max",
					Value: cmDuration,
				}
				metricsservice.WriteBluemixMetrics([]metricsservice.BluemixMetric{bm}, true, measurement, "")

				if err != nil {
					fmt.Println("ERROR: Listing namespaces", err)
					listError = true
				} else {
					for _, namespace := range namespaces.Items {
						if namespaceRegex.MatchString(namespace.Name) {
							cruiserName := strings.TrimPrefix(namespace.Name, "master-")

							if _, ok := foundCruisers[cruiserName]; ok == false {
								foundCruisers[cruiserName] = extractKubeConfig(namespace.Name, cruiserName)
							} else {
								foundCnt++
								foundCruisers[cruiserName] = true
							}
						}
					}

				}
			}
			// Get hypershift cluster secrets
			timeStart = time.Now()
			secrets, err = carrier.GetSecretsByNamespace("master")
			timeEnd = time.Now()
			cmDuration = timeEnd.Sub(timeStart)

			metricsPrefix = strings.Join([]string{measurement}, ".")
			bm = metricsservice.BluemixMetric{
				Name:  metricsPrefix + ".secret_list.Latency.max",
				Value: cmDuration,
			}
			metricsservice.WriteBluemixMetrics([]metricsservice.BluemixMetric{bm}, true, measurement, "")

			if err != nil {
				fmt.Println("ERROR: Listing Hypershift secrets", err)
				listError = true
			} else {
				for _, secret := range secrets.Items { // pragma: allowlist secret
					if hypershiftSecretRegex.MatchString(secret.Name) {
						cruiserName := strings.TrimSuffix(secret.Name, "-admin-kubeconfig")

						if _, ok := foundCruisers[cruiserName]; ok == false {
							foundCruisers[cruiserName] = extractKubeConfig("master", cruiserName)
						} else {
							foundCnt++
							foundCruisers[cruiserName] = true
						}
					}
				}
			}

			if !listError && foundCnt < len(foundCruisers) {
				for name := range foundCruisers {
					if foundCruisers[name] == false {
						delete(foundCruisers, name)
						cruiserCrud <- Cruiser{name: name, state: Deleted}
						if debug {
							fmt.Printf("Removing deleted configuration: %s\n", name)
						}
					}
				}
			}
		}
		waitForLoopDuration("Cruiser config", loopStart)
		firstIteration = false
	}
}

// createKubeConfigDir ...
func createKubeConfigDir(configBase string) error {
	// Store cruiser kubernetes configuration locally
	if _, err := os.Stat(configBase); os.IsNotExist(err) {
		err := os.MkdirAll(configBase, os.ModePerm)
		if err != nil {
			// Bad situation which will likely continue to happen, but never should
			fmt.Printf("ERROR: Creating directory %s: %v\n", configBase, err)
			return err
		}
	}
	return nil
}

// extractKubeConfig ...
func extractKubeConfig(namespace string, cruiserName string) bool {
	var foundCruiser bool
	var deploymentName string
	configBase := cruiserConfigPath + "/" + cruiserName
	config := configBase + "/admin-kubeconfig"

	if namespace == "kubx-masters" {
		configName := "master-" + cruiserName + "-config"
		configMap, err := carrier.GetConfigMap(namespace, configName)
		if err != nil {
			fmt.Println("WARNING: Couldn't find cert config map: ", configName)
			return false
		}

		err = createKubeConfigDir(configBase)
		if err != nil {
			return false
		}

		err = storeConfigMapToFile(configMap.Data, "admin-kubeconfig", config)
		if err != nil {
			fmt.Println("Error: Failed to setup kubeconfig", err)
		}
		deploymentName = "master-" + cruiserName
	} else if namespace == "master" {
		certName := cruiserName + "-admin-kubeconfig"
		certData, err := carrier.GetSecret(namespace, certName)
		if err != nil {
			fmt.Println("WARNING: Couldn't find Hypershift admin-kubeconfig secret: ", certName) // pragma: allowlist secret
			return false
		}

		err = createKubeConfigDir(configBase)
		if err != nil {
			return false
		}

		storeSecretToFile(certData.Data, "kubeconfig", config)
		deploymentName = "kube-apiserver"

	} else {
		certName := "openvpn-operator-secret"
		certData, err := carrier.GetSecret(namespace, certName)
		if err != nil {
			fmt.Println("WARNING: Couldn't find OpenShift openvpn-operator-secret secret: ", certName) // pragma: allowlist secret
			return false
		}

		err = createKubeConfigDir(configBase)
		if err != nil {
			return false
		}

		storeSecretToFile(certData.Data, "admin-kubeconfig", config)
		deploymentName = "kube-apiserver"
	}

	err := setupKube(namespace, cruiserName, config, deploymentName)
	if err == nil {
		foundCruiser = true
	}

	return foundCruiser
}

// setupKube ...
func setupKube(namespace string, cruiserName string, config string, deploymentName string) error {
	var err error
	if _, err = os.Stat(config); os.IsNotExist(err) == false && os.IsPermission(err) == false {
		// This statement can take 4 seconds to respond
		cube, err := kube.CreateKube(config, timeout)
		if err != nil {
			fmt.Printf("ERROR: Trying to initialize kube API with %s: %v\n", config, err)
			return err
		}

		replicas, err := carrier.GetDeploymentReplicas(namespace, deploymentName)
		if err != nil {
			// Just assume ha cruisers are being used.
			replicas = 3
			err = nil
		}
		cruiser := Cruiser{name: cruiserName, state: Booting, cube: cube, discovered: time.Now(), replicas: replicas, namespace: namespace}
		if debug {
			fmt.Printf("Found %s\n", cruiserName)
		}
		cruiserCrud <- cruiser
	} else {
		err = errors.New("ERROR: Kubeconfig file doesn't exist or isn't readable: " + config)
		if debug {
			fmt.Printf("Kubeconfig file doesn't exist or isn't accessible %s\n", config)
		}
	}
	return err
}

// storeConfigMapToFile ...
func storeConfigMapToFile(configMapData map[string]string, key string, fileName string) error {
	var err error
	if data, ok := configMapData[key]; ok {
		var c *os.File
		if c, err = os.Create(fileName); err == nil {
			c.Chmod(0600)
			b := bytes.NewBufferString(data)
			if _, err = io.Copy(c, b); err == nil {
				c.Sync()
			}
			c.Close()
		} else {
			err = errors.New("ERROR: Failed to create config file " + fileName)
		}
	} else {
		err = errors.New("ERROR: config map doesn't contain " + key)
	}

	return err
}

// storeSecretToFile ...
func storeSecretToFile(secretMapData map[string][]byte, key string, fileName string) error {
	var err error

	if data, ok := secretMapData[key]; ok {
		var c *os.File
		if c, err = os.Create(fileName); err == nil {
			c.Chmod(0600)
			// This two step process decodes the secret
			j := bytes.NewBuffer(data)
			b := bytes.NewBufferString(j.String())
			if _, err = io.Copy(c, b); err == nil {
				c.Sync()
			}
			c.Close()
		} else {
			err = errors.New("ERROR: Failed to create config file " + fileName)
		}
	} else {
		err = errors.New("ERROR: config map doesn't contain " + key)
	}

	return err
}
