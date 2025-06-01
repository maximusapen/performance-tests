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
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/config"
	"github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/request"
	"github.ibm.com/alchemy-containers/armada-performance/api/cloud_mon/lib/monitor"
)

func main() {
	var clusterPrefix string // Prefix to be used to Cruiser names
	var numThreads int       // Number of concurrent API requests
	var totalUsers int       // Total unique organizations
	var totalClusters int    // Total number of cruisers per user
	var totalWorkers int     // Total number of workers per user
	var machineType string   // Machine type, "free", "small", "medium", "large", "xlarge", "xxlarge"
	var debug bool           // Output http request/response summary
	var verbose bool         // Output http request/response details
	var backgroundApp string // Application running on each clusters node for connectivity testing (1 per node)
	var activeApp string
	var tests string

	flag.StringVar(&clusterPrefix, "clusterNamePrefix", "cloudMonitor", "")
	flag.IntVar(&numThreads, "numThreads", 0, "number of concurrent requests")
	flag.IntVar(&totalUsers, "users", 1, "total number of users")
	flag.IntVar(&totalClusters, "clusters", 1, "total number of clusters per user")
	flag.IntVar(&totalWorkers, "workers", 3, "total number of worker nodes")
	flag.StringVar(&machineType, "machineType", "u1c.2x4", "machine type: free or u1c.2x4/b1c.4x16/b1c.16x64/b1c.32x128/b1c.56x242")
	flag.BoolVar(&verbose, "verbose", false, "output request and response summary")
	flag.BoolVar(&debug, "debug", false, "debug logging output")
	flag.StringVar(&backgroundApp, "backgroundApp", "etcdcm", "Name of app use for cluster testing")
	flag.StringVar(&activeApp, "activeApp", "cloud-mon-app", "Name of app created each time through loop")
	var testPattern string
	for i := 0; i < monitor.NumberOfTests(); i++ {
		if i == 0 {
			testPattern = "y"
		} else {
			testPattern = testPattern + ",y"
		}
	}
	flag.StringVar(&tests, "tests", testPattern, "Signals which tests should be run")

	flag.Parse()

	if numThreads == 0 {
		// Default to processing ALL requests in parallel
		numThreads = totalUsers * totalClusters
	}

	// If not explicity specified, default machine type based on number of clusters/workers
	// (A free "patrol" account is limited to 1 cluster and worker)
	if len(machineType) == 0 {
		if totalWorkers > 1 || totalClusters > 1 {
			machineType = "small"
		} else {
			machineType = config.FreeAccountStr
		}
	}

	var testsToRun = make([]bool, monitor.NumberOfTests())
	testsString := strings.Split(tests, ",")
	if len(testsString) != monitor.NumberOfTests() {
		fmt.Println("Must specify a value for each of the ", monitor.NumberOfTests(), " tests. Ex -tests "+strings.Replace(testPattern, ",y", ",n", 1))
		os.Exit(1)
	}

	for j, tst := range testsString {
		testsToRun[j] = tst == "y"
	}

	if verbose {
		monitor.Debug = true
	}

	basePath := config.GetConfigPath()
	var conf config.Config
	config.ParseConfig(filepath.Join(basePath, "perf.toml"), &conf)

	requestJobs := make(chan request.Data, totalUsers*totalClusters)
	requestCompleted := make(chan request.Data, totalUsers*totalClusters)
	request.InitRequests(machineType, "", "", &conf, verbose, debug, false, nil, nil, numThreads, &requestJobs, &requestCompleted)

	// start the test
	monitor.Run(&conf, clusterPrefix, totalWorkers, activeApp, backgroundApp, testsToRun)
}
