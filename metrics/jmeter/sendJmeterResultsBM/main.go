/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2018, 2021 All Rights Reserved.
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
	metrics         bool
	verbose         bool
	multizone       bool
	resultsFile     string
	testName        string
	dbKey           string
	numThreads      string
	metricsTestName string
)

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

	// Read the last line which has the summary data
	lastLine := results[len(results)-1]
	numCols := len(lastLine)

	fmt.Println("Last line of results csv file has", numCols, "fields")

	if numCols < 11 {
		fmt.Println("Error in last line of ${resultsFile}. No metrics will be send to the metrics service.")
		return
	}

	total, _ := strconv.ParseFloat(lastLine[1], 32)
	average, _ := strconv.ParseFloat(lastLine[2], 32)
	percentile99, _ := strconv.ParseFloat(lastLine[6], 32)
	errorPercent := lastLine[9]
	//strip off the percent sign
	errors, _ := strconv.ParseFloat(errorPercent[:len(errorPercent)-1], 32)
	throughput, _ := strconv.ParseFloat(lastLine[10], 32)

	fmt.Println(total, average, percentile99, errorPercent, throughput, time.Now().Unix())

	// Jmeter reports an invalid throughput result if there are errors reported
	if errors > 0 {
		fmt.Println("The error percentage is greater than zero so setting throughput", throughput, "to zero.")
		throughput = 0
	}

	// populate the metrics
	threadMetrics := "threads_" + numThreads
	var clusterZoneType string
	if multizone {
		clusterZoneType = "multizone"
	} else {
		clusterZoneType = "singlezone"
	}

	bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{metricsTestName, threadMetrics, clusterZoneType, "total", "sparse-avg"}, "."), Timestamp: time.Now().Unix(), Value: total})
	bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{metricsTestName, threadMetrics, clusterZoneType, "average", "sparse-avg"}, "."), Timestamp: time.Now().Unix(), Value: average})
	bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{metricsTestName, threadMetrics, clusterZoneType, "percentile99", "sparse-avg"}, "."), Timestamp: time.Now().Unix(), Value: percentile99})
	bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{metricsTestName, threadMetrics, clusterZoneType, "errorPercentage", "sparse-avg"}, "."), Timestamp: time.Now().Unix(), Value: errors})
	bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{metricsTestName, threadMetrics, clusterZoneType, "throughput", "sparse-avg"}, "."), Timestamp: time.Now().Unix(), Value: throughput})
	// Needed to have sparse-avg in the metric for retention policy roll up

	if verbose {
		for _, m := range bm {
			fmt.Println(m)
		}
	}

	if metrics {
		metricsservice.WriteBluemixMetrics(bm, true, testName, dbKey)
	}

	if errors > 0 {
		log.Fatalln("The error percentage is greater than zero")
	}
}

// Main - sends the output from the CSV to the metrics service

func main() {
	log.SetOutput(os.Stdout)

	flag.StringVar(&testName, "testname", "", "Test name in Jenkins - only needed if sending alerts to RazeeDash")
	flag.StringVar(&metricsTestName, "metricsTestName", "", "Specific name to be used in metrics path i.e HTTPNodePort.nodes5_replica3 rather than the overall test name - httperf")
	flag.StringVar(&dbKey, "dbkey", "", "Metrics database key - only needed if sending metrics to database")
	flag.BoolVar(&verbose, "verbose", true, "verbose output")
	flag.BoolVar(&metrics, "metrics", false, "send results to IBM Metrics service")
	flag.BoolVar(&multizone, "multizone", false, "multizone environment")
	flag.StringVar(&resultsFile, "resultsFile", "", "Results file to parse")
	flag.StringVar(&numThreads, "numThreads", "", "Number of threads tested")
	flag.Parse()

	fmt.Printf("result file is %s\n", resultsFile)

	if len(dbKey) == 0 {
		dbKey = os.Getenv("METRICS_DB_KEY")
	}

	writeMetrics()
}
