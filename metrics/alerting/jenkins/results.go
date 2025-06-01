/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2021, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package jenkins

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.ibm.com/alchemy-containers/armada-performance/metrics/alerting/config"
)

// Builds Contains a list of jenkins builds
type Builds struct {
	Timestamp int64   `json:"timestamp"`
	Builds    []Build `json:"builds"`
}

// Build Data from a single jenkins build
type Build struct {
	Result           string       `json:"result"`
	Description      string       `json:"description"`
	URL              string       `json:"url"`
	Number           int          `json:"number"`
	Timestamp        int64        `json:"timestamp"`
	ClusterPrefix    string       `json:"cluster_prefix"`
	ClusterType      string       `json:"cluster_type"`
	PerfClients      string       `json:"perf_clients"`
	PerfTest         string       `json:"perf_test"`
	K8sVersion       string       `json:"k8s_version"`
	WorkerType       string       `json:"worker_type"`
	CloudEnvironment string       `json:"cloud_environment"`
	Zones            string       `json:"zones"`
	Type             string       `json:"type"`
	Success          bool         `json:"success"`
	Highlight        bool         `json:"highlight"`
	ChartURL         string       `json:"charturl"`
	CarrierName      string       `json:"carriername"`
	ClusterEnvName   string       `json:"clusterenvname"`
	DayOffset        int          `json:"dayoffset"`
	Day              time.Weekday `json:"day"`
	DeleteCluster    bool         `json:"delete_cluster"`
}

// TestInfo holds relevant information about an automatin test
type TestInfo struct {
	Name        string
	Timestamp   int64
	KubeVersion string
	URL         string
}

// FailureData holds data for test case failures
type FailureData struct {
	Count int
	Tests []TestInfo
}

// Failures returns a list of failed test cases for each environment
func Failures(conf *config.Data) map[string]*FailureData {
	// Process automation results file
	var r Builds

	// On Mondays we need to look at the last 3 days results
	var daysToProcess int64 = 1
	weekday := time.Now().Weekday()
	if weekday == time.Monday {
		daysToProcess = 3
	}
	validPeriod := int64(time.Hour) * 24 * daysToProcess / int64(time.Second)
	validTime := (time.Now().Unix() - validPeriod) * 1000

	failures := make(map[string]*FailureData)
	for e := range conf.Environments {
		failures[e] = &FailureData{}
	}

	if !conf.Options.Failures {
		return failures
	}

	f, err := os.Open("parseJenkinsResults.builds.json")
	if err != nil {
		log.Printf("Uable to open automation results json file : %s\n", err)
		return failures
	}

	j := json.NewDecoder(f)
	err = j.Decode(&r)
	if err != nil {
		log.Fatalf("Error parsing json : %s", err)
	}

	for _, b := range r.Builds {
		// Ignore test results from earlierreporting periods
		if validTime < b.Timestamp {
			if b.Result == "FAILURE" {
				var envName string

				switch b.ClusterType {
				case "Classic":
					if strings.Contains(b.K8sVersion, "openshift") {
						envName = "ROKS on Classic"
					} else {
						envName = "IKS on Classic"
					}
				case "VPC-Gen2":
					if strings.Contains(b.K8sVersion, "openshift") {
						envName = "ROKS on VPC"
					} else {
						envName = "IKS on VPC"
					}
				case "Satellite":
					envName = "Satellite"
				default:
					// Unrecognized environment - skip
					continue
				}

				failures[envName].Count++
				ti := TestInfo{Name: b.PerfTest, Timestamp: b.Timestamp, KubeVersion: b.K8sVersion, URL: b.URL}
				if ti.Name == "" {
					if b.DeleteCluster {
						ti.Name = "Cluster Cleanup"
					} else {
						ti.Name = "Cluster Setup"
					}
				}
				failures[envName].Tests = append(failures[envName].Tests, ti)
			}
		}
	}
	return failures
}
