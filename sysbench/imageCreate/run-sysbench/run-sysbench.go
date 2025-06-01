/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2020 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	metricsservice "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"
)

var (
	metrics     bool
	verbose     bool
	resultsFile string
	testName    string
)

// Run the execute_sysbench shell script which writes output to a CSV file
func runSysbench() {
	shellCommand := "./executeSysbench.sh"
	// #nosec G204
	cmd, err := exec.Command(shellCommand, resultsFile).CombinedOutput()
	if err != nil {
		log.Fatalln(err.Error())
	}
	fmt.Printf("cmd output is %s\n", cmd)
}

// Take the output of the CSV file and send to the metrics service
func writeMetrics() {
	if len(resultsFile) == 0 {
		log.Fatalln("csv file not specified")
	}
	// #nosec G204
	cmd := exec.Command("cat", resultsFile)
	err := cmd.Run()
	if err != nil {
		log.Fatalln(err.Error())
	}

	// Open the CSV results file and read it
	// #nosec G304
	resultsReader, err := os.Open(resultsFile)
	if err != nil {
		log.Fatalln(err.Error())
	}

	csvReader := csv.NewReader(bufio.NewReader(resultsReader))
	csvReader.FieldsPerRecord = -1

	results, err := csvReader.ReadAll()
	if err != nil {
		log.Fatalln(err.Error())
	}

	var bm = []metricsservice.BluemixMetric{}

	for i, record := range results[1:] {
		fmt.Println("Record", i, "has", len(record), "fields")

		//hostName := record[1]
		//machineType := record[2]
		worker := record[3]
		testType := record[4]
		numThreads := record[5]
		min, _ := strconv.ParseFloat(record[7], 2)
		avg, _ := strconv.ParseFloat(record[8], 2)
		max, _ := strconv.ParseFloat(record[9], 2)
		percentile95, _ := strconv.ParseFloat(record[10], 2)
		// fmt.Println(hostName, testType, min, max, percentile95)

		//bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{"sysbench", testType, "threads_" + numThreads, machineType, worker, "min"}, "."), Timestamp: time.Now().Unix(), Value: min})
		bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{"sysbench", testType, "threads_" + numThreads, worker, "min", "sparse-avg"}, "."), Timestamp: time.Now().Unix(), Value: min})
		//bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{"sysbench", testType, "threads_" + numThreads, machineType, worker, "max"}, "."), Timestamp: time.Now().Unix(), Value: max})
		// Need to have sparse-avg in the metric for retention policy.
		bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{"sysbench", testType, "threads_" + numThreads, worker, "max", "sparse-avg"}, "."), Timestamp: time.Now().Unix(), Value: max})
		//bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{"sysbench", testType, "threads_" + numThreads, machineType, worker, "avg"}, "."), Timestamp: time.Now().Unix(), Value: avg})
		bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{"sysbench", testType, "threads_" + numThreads, worker, "avg", "sparse-avg"}, "."), Timestamp: time.Now().Unix(), Value: avg})
		//bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{"sysbench", testType, "threads_" + numThreads, machineType, worker, "95thpercentile"}, "."), Timestamp: time.Now().Unix(), Value: percentile95})
		bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{"sysbench", testType, "threads_" + numThreads, worker, "95thpercentile", "sparse-avg"}, "."), Timestamp: time.Now().Unix(), Value: percentile95})
	}
	if verbose {
		for _, m := range bm {
			fmt.Println(m)
		}
	}

	if metrics {
		//Running in a cruiser so don't need to send dbkey
		metricsservice.WriteBluemixMetrics(bm, true, testName, "")
	}
}

// Main - runs the sysbench shellscript and then sends the output from the CSV to the metrics service

func main() {
	log.SetOutput(os.Stdout)

	flag.StringVar(&testName, "testname", "", "Test name in Jenkins - only needed if sending alerts to RazeeDash")
	flag.BoolVar(&verbose, "verbose", true, "verbose output")
	flag.BoolVar(&metrics, "metrics", false, "send results to IBM Metrics service")
	flag.Parse()
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "ahost"
	}

	// Set the name for the output CSV
	resultsFile = "sysbench_results_" + hostname + ".csv"
	fmt.Printf("result file is %s\n", resultsFile)

	runSysbench()
	writeMetrics()

	//Don't want the go program to exit yet as the daemonset will restart
	//so Keep alive
	select {}
}
