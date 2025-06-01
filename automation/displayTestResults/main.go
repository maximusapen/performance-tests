/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2020, 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

/*
 * The Generate-Results-Visualization jenkins job calls parseJenkinsResults.sh to get data
 * on the recent Run-Performance-Tests builds (/tmp/parseJenkinsResults.builds.json). This program takes that list
 * extracts additional data from jenkins and generates a table showing the results of the automation
 * tests for the last week. The results are viewable via the artifacts of each Generate-Results-Visualization build.
 *
 * This program builds `results`, which contains ClusterEnvs (i.e. carrier, kube version, cluster type) that are the
 * headers for the main table, and the subject of subsequent tables. `results` also contains Builds which are organized
 * to align with ClusterEnvs and represents the data rows of the table. This setup is needed so that the template can
 * easily walk through the data and generate html.
 */

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"time"
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
	OperatingSystem  string       `json:"operating_system"`
}

// ClusterEnv Decribes the cluster as well as the environment in which it is run (i.e. <type> <kube version> <carrier> -> Classic 1.17 carrier4_stage)
type ClusterEnv struct {
	Name            string
	ClusterType     string
	ChartURL        string
	K8sVersion      string
	WorkerType      string
	CarrierName     string
	Zones           string
	Day             time.Weekday
	Type            string
	OperatingSystem string
}

// Test Builds associated with a particular test
type Test struct {
	Name   string
	Builds []Build
}

// Results Tests stores a matrix of builds which maps to the html tables to be generated
type Results struct {
	ClusterEnvs []ClusterEnv `json:"environments"`
	Tests       []Test       `json:"tests"`
}

var results Results
var clusterEnvNames []string

var weekdays = []time.Weekday{
	time.Sunday,
	time.Monday,
	time.Tuesday,
	time.Wednesday,
	time.Thursday,
	time.Friday,
	time.Saturday,
}
var automationSetup = "cluster setup"
var automationCleanup = "cluster cleanup"

const dayInMilliseconds = 24 * 60 * 60 * 1000
const weekInMilliseconds = 7 * dayInMilliseconds

var summaryTableTemplate = `<table>
<tr><td>Test</td>{{range .ClusterEnvs}}
<td><a href="#{{.Name}}">{{if .K8sVersion}}{{.K8sVersion}}<br>{{.ClusterType}}<br>{{.Zones}}<br>{{.CarrierName}}{{if .OperatingSystem}}<br>{{.OperatingSystem}}{{end}}{{else}}Registry{{end}}
</a><br>
<a href="{{.ChartURL}}">
<img src="line-chart.png" alt="Grafana Charts" style="width:20px;height:20px;"/>
</a>
</td>
{{end}}</tr>
	{{range .Tests}}<tr>
<td>{{.Name}}</td>{{range .Builds}}    <td>{{if .Result}}
<a href="{{.URL}}console">
	   {{if .Highlight}}
{{if .Success}}
<span style="color:green">{{.Result}}</span>
{{else}}
<span style="color:red">{{.Result}}</span>
{{end}}
{{else}}
<span style="color:black">{{.Result}}</span>
{{end}}
</a>
{{if .ChartURL}}
<a href="{{.ChartURL}}">
<img src="https://img.icons8.com/wired/64/000000/line-chart.png" alt="Grafana Charts" style="width:20px;height:20px;"/>
</a>
{{end}}
{{end}}
</td>
{{end}}</tr>
{{end}}
</table>`

var environmentTableTemplate = `<br>
<style>table { border-collapse: collapse; }
table, th, td { border: 1px solid black; }
tr:nth-child(even) {background-color: #f2f2f2;}
tr:hover {background-color:#d6d6d6;}
</style>
<table>
<tr><td>Test</td>{{range .ClusterEnvs}}
<td>{{.Name}}
</td>
{{end}}</tr>
	{{range .Tests}}<tr>
<td>{{.Name}}</td>{{range .Builds}}    <td>{{if .Result}}
<a href="{{.URL}}console">
	   {{if .Highlight}}
{{if .Success}}
<span style="color:green">{{.Result}}</span>
{{else}}
<span style="color:red">{{.Result}}</span>
{{end}}
{{else}}
<span style="color:black">{{.Result}}</span>
{{end}}
</a>
{{if .ChartURL}}
<a href="{{.ChartURL}}">
<img src="https://img.icons8.com/wired/64/000000/line-chart.png" alt="Grafana Charts" style="width:20px;height:20px;"/>
</a>
{{end}}
{{end}}
</td>
{{end}}</tr>
{{end}}
</table>`

func findClusterEnv(name string) int {
	for index, clusterEnv := range results.ClusterEnvs {
		if clusterEnv.Name == name {
			return index
		}
	}
	return -1
}

func generateDefaultTestBuilds(clusterEnvs []ClusterEnv) []Build {
	var builds []Build
	for _, clusterEnv := range clusterEnvs {
		build := Build{PerfTest: clusterEnv.Name}
		builds = append(builds, build)
	}
	return builds
}

func generateCalendarWeekTestBuilds() []Build {
	var builds []Build
	for _, weekday := range weekdays {
		build := Build{PerfTest: weekday.String()}
		builds = append(builds, build)
	}
	return builds
}

func findTest(name string) int {
	for index, test := range results.Tests {
		if test.Name == name {
			return index
		}
	}
	return -1
}

func findBuild(build Build) int {
	for index, clEnv := range results.ClusterEnvs {
		if clEnv.Name == build.ClusterEnvName {
			return index
		}
	}
	return -1
}

func findBuildByWeekday(build Build) int {
	for index, clEnv := range results.ClusterEnvs {
		if clEnv.Name == build.Day.String() {
			return index
		}
	}
	return -1
}

func populateResultTests(tests []string, builds []Build) {
	for _, testName := range tests {
		test := Test{Name: testName}
		for i := range builds {
			test.Builds = append(test.Builds, builds[i])
		}
		results.Tests = append(results.Tests, test)
	}
}

func generateDefaultClusterEnvBuilds(tests []string) []Build {
	var builds []Build
	for _, testName := range tests {
		build := Build{PerfTest: testName, Result: "DUMMY"}
		builds = append(builds, build)

	}
	return builds
}

func generateClusterEnvName(build Build) string {
	var clusterEnvName string
	if build.Type == "registry" {
		clusterEnvName = "registry"
	} else {
		clusterEnvName = build.K8sVersion
		if build.ClusterType != "Classic" {
			clusterEnvName = clusterEnvName + " " + build.ClusterType
		}
		if build.CarrierName != "" {
			clusterEnvName = clusterEnvName + " " + build.CarrierName
		}
		if build.Zones != "dal09" {
			clusterEnvName = clusterEnvName + " " + build.Zones
		}
		if build.OperatingSystem != "" {
			clusterEnvName = clusterEnvName + " " + build.OperatingSystem
		}
	}
	return clusterEnvName
}

func getDaysOffset(build Build, extractTimestamp int64, weekday time.Weekday) (int, time.Weekday) {
	dayOffset := (extractTimestamp - build.Timestamp) / dayInMilliseconds
	testweekday := int(weekday) - 1 - int(dayOffset)
	if testweekday < 0 {
		testweekday = testweekday + 7
	}
	return int(dayOffset), time.Weekday(testweekday)
}

func getCarrierName(build Build) string {
	if build.CloudEnvironment == "Production" {
		return "us-south_prod"
	}
	perfClientSplit := strings.SplitAfter(build.PerfClients, "-")
	carrierName := "carrier" + perfClientSplit[2][4:5] + "_" + perfClientSplit[0][0:5]
	return carrierName
}

func buildTableElements(builds Builds, htmlTemplate string, cutoffTimestamp int64, highlightTimestamp int64, clusterEnvFilter string, summaryTable bool) {
	// Process each build by:
	// - Filter unwanted builds
	// - Enrich builds
	// - place build in the results matrix
	for _, build := range builds.Builds {
		//fmt.Println(build.URL)

		if build.Timestamp < cutoffTimestamp {
			break
		}
		// Add filter for builds
		if clusterEnvFilter != "" && build.ClusterEnvName != clusterEnvFilter {
			continue
		}
		//fmt.Println("Found ", build.ClusterEnvName)

		build.Highlight = build.Timestamp > highlightTimestamp

		build.Success = build.Result == "SUCCESS"

		if !build.Highlight {
			build.Result = strings.ToLower(build.Result)
		}

		var index int
		if build.Type == "registry" {
			// TODO what about "Registry-Load"?
			index = findTest("Registry")
		} else {
			index = findTest(build.PerfTest)
		}
		if index == -1 {
			fmt.Fprintln(os.Stderr, build.Type)
			fmt.Fprintln(os.Stderr, results.Tests)
			fmt.Fprintln(os.Stderr, "Failed build data for environemnt tables", build)
			panic("Unexpected test")
		}
		var buildIndex int
		if summaryTable {
			buildIndex = findBuild(build)
		} else {
			buildIndex = findBuildByWeekday(build)
		}

		// Only display the latest result for a test
		if results.Tests[index].Builds[buildIndex].Result == "" || results.Tests[index].Builds[buildIndex].Timestamp < build.Timestamp {
			//fmt.Fprintln(os.Stderr, "Overwriting existing build (current, replacement):", results.Tests[index].Builds[buildIndex].Timestamp, build.Timestamp)

			/* TODO pull test specific URL from test.json, Don't do until cell tells whether test results passed/failed
			kVersion := strings.ReplaceAll(build.K8sVersion, "_openshift", "")
			kVersion = strings.ReplaceAll(kVersion, ".", "_")

			build.ChartURL = "https://alchemy-dashboard.containers.cloud.ibm.com/stage/performance/grafana/d/jaPhwBemk/performance-summary?orgId=1&var-CarrierName=" + build.CarrierName + "&var-MachineType=" + build.WorkerType + "&var-KubeVersion=" + kVersion
			*/

			results.Tests[index].Builds[buildIndex] = build
		}
	}

	// Generate the html
	t := template.New("t")
	t, err := t.Parse(htmlTemplate)
	if err != nil {
		panic(err)
	}

	err = t.Execute(os.Stdout, results)
	if err != nil {
		panic(err)
	}
}

// TODO Parse logs for failed builds for common failures like can't connect to perf client. "/consoleText"
func main() {
	// Alphabetized list of tests. The order in the array is the order in the display
	// Note: If a new automation test is added then this table must be updated for the tests results to show in the visualization
	tests := []string{automationSetup, automationCleanup, "APIServer-Load", "Acmeair", "Acmeair-image", "Acmeair-istio", "Acmeair-istio-extras", "Acmeair-istio-image", "Add-Workers", "http-perf", "http-scale", "https-perf", "K8s-E2e-Performance-Density", "K8s-E2e-Performance-Load", "K8s-Netperf", "Kubemark", "Node-Autoscaler", "OLB-Java", "Persistent-Storage", "Pod-Scaling", "Registry", "Registry-load", "Snapshot-Storage", "Sysbench", "ZeroWorkerClusters", "incluster-apiserver", "odf-storage", "odf-storage-parallel", "odf-storage-nodisable", "armada-api-load", "portworx-storage", "portworx-storage-parallel", "portworx-storage-nodisable"}

	buildsFile, err := os.Open("/tmp/parseJenkinsResults.builds.json")

	if err != nil {
		fmt.Println(err)
	}
	defer buildsFile.Close()

	fmt.Println("<html><body>")
	fmt.Println("<a href=\"./schedule.html\">Automation schedule</a><br><br>")
	timeStamp := time.Now().Format("Mon Jan 02 15:04:05 MST 2006")
	fmt.Println("<b>Report generated</b>:", timeStamp, "<br>")

	buildsBytes, _ := ioutil.ReadAll(buildsFile)

	var builds Builds

	json.Unmarshal(buildsBytes, &builds)

	// Filter out anything older than 7 days
	cutoffTimestamp := builds.Timestamp - weekInMilliseconds

	// On Mondays we need to look at the last 3 days results
	daysToHighlight := 1
	weekday := time.Now().Weekday()
	if weekday == time.Monday {
		daysToHighlight = 3
	}

	highlightTimestamp := builds.Timestamp - (int64(daysToHighlight) * dayInMilliseconds)

	// Create ClusterEnvs (i.e. columns) based on the builds, and enrich data in builds
	for i, build := range builds.Builds {
		if build.Timestamp < cutoffTimestamp {
			break
		}

		if build.PerfTest == "" {
			if build.DeleteCluster {
				build.PerfTest = automationCleanup
			} else {
				build.PerfTest = automationSetup
			}
		}

		build.DayOffset, build.Day = getDaysOffset(build, builds.Timestamp, weekday)
		clusterEnvName := generateClusterEnvName(build)
		build.ClusterEnvName = clusterEnvName
		build.CarrierName = getCarrierName(build)
		builds.Builds[i] = build

		// Ignore builds when there is alread a ClusterEnv
		clusterEnvIndex := findClusterEnv(clusterEnvName)
		if clusterEnvIndex == -1 {
			kVersion := strings.ReplaceAll(build.K8sVersion, "_openshift", "")
			kVersion = strings.ReplaceAll(kVersion, ".", "_")

			clusterEnv := ClusterEnv{Name: clusterEnvName, K8sVersion: build.K8sVersion, WorkerType: build.WorkerType, CarrierName: build.CarrierName, ClusterType: build.ClusterType, Zones: build.Zones, Day: builds.Builds[i].Day, OperatingSystem: build.OperatingSystem}

			// Construct the links to the charts
			if build.Type == "registry" {
				clusterEnv.ChartURL = "https://alchemy-dashboard.containers.cloud.ibm.com/stage/performance/grafana/d/zIduDHnWz/registry?orgId=1&var-TestName=registry&var-CarrierName=carrier5_stage&var-CarrierName=carrier3_stage"
				build.ChartURL = clusterEnv.ChartURL
			} else {
				build.ChartURL = "https://alchemy-dashboard.containers.cloud.ibm.com/stage/performance/grafana/d/jaPhwBemk/performance-summary?orgId=1"
				if build.ClusterType == "VPC-Gen2" {
					if strings.Contains(build.K8sVersion, "openshift") {
						clusterEnv.ChartURL = "https://alchemy-dashboard.containers.cloud.ibm.com/stage/performance/grafana/d/Z3oFo3mMz/performance-summary-roks-v4-vpc-gen2?orgId=1"
					} else {
						clusterEnv.ChartURL = "https://alchemy-dashboard.containers.cloud.ibm.com/stage/performance/grafana/d/M0THqf6Wk/performance-summary-vpc-gen2?orgId=1"
					}
				} else {
					if strings.Contains(build.K8sVersion, "openshift") {
						clusterEnv.ChartURL = "https://alchemy-dashboard.containers.cloud.ibm.com/stage/performance/grafana/d/xqJchY6Wz/performance-summary-roks-v4?orgId=1"
					} else {
						clusterEnv.ChartURL = "https://alchemy-dashboard.containers.cloud.ibm.com/stage/performance/grafana/d/jaPhwBemk/performance-summary?orgId=1"
					}
				}
				workerType := strings.ReplaceAll(build.WorkerType, ".", "_")
				operatingSystemClause := ""
				if build.OperatingSystem != "" {
					// Only set the OS variable if we have a value. This way
					// if no value is set we will use the pages default instead.
					operatingSystemClause = "&var-OperatingSystem=" + strings.ReplaceAll(build.OperatingSystem, "-", "_")
				}
				build.ChartURL = build.ChartURL + "&var-CarrierName=" + build.CarrierName + "&var-MachineType=" + workerType + "&var-KubeVersion=" + kVersion + operatingSystemClause
				clusterEnv.ChartURL = clusterEnv.ChartURL + "&var-CarrierName=" + build.CarrierName + "&var-MachineType=" + workerType + "&var-KubeVersion=" + kVersion + operatingSystemClause
			}

			results.ClusterEnvs = append(results.ClusterEnvs, clusterEnv)
		}
	}

	sort.Slice(results.ClusterEnvs, func(i, j int) bool { return results.ClusterEnvs[i].Name < results.ClusterEnvs[j].Name })
	for _, clusterEnv := range results.ClusterEnvs {
		clusterEnvNames = append(clusterEnvNames, clusterEnv.Name)
	}

	// Generate the default row of builds
	defaultTestBuilds := generateDefaultTestBuilds(results.ClusterEnvs)

	// Pre populate results.Tests so there aren't any gaps in table
	populateResultTests(tests, defaultTestBuilds)

	// Generate the html for the primary test results table
	buildTableElements(builds, summaryTableTemplate, cutoffTimestamp, highlightTimestamp, "", true)

	// Create a separate html table for each ClusterEnv
	for _, clusterEnvFilter := range clusterEnvNames {
		fmt.Printf("<a id=\"%s\"><b>%s</b></a>", clusterEnvFilter, clusterEnvFilter)

		defaultTestBuilds = generateCalendarWeekTestBuilds()
		results.ClusterEnvs = nil
		results.Tests = nil
		for _, weekday := range weekdays {
			clusterEnv := ClusterEnv{Name: weekday.String()}
			results.ClusterEnvs = append(results.ClusterEnvs, clusterEnv)
		}

		populateResultTests(tests, defaultTestBuilds)

		buildTableElements(builds, environmentTableTemplate, cutoffTimestamp, highlightTimestamp, clusterEnvFilter, false)
	}
	fmt.Println("</body></html>")

}
