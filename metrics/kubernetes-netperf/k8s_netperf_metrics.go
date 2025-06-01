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
	"strconv"
	"strings"
	"time"

	metricsservice "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"
)

var (
	metrics     bool
	verbose     bool
	resultsFile string
)

func main() {
	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.BoolVar(&metrics, "metrics", false, "send results to IBM Metrics service")
	flag.StringVar(&resultsFile, "resultsFile", "", "Kubernetes netperf csv results filename")
	flag.Parse()

	if len(resultsFile) == 0 {
		log.Fatalln("Please specify -resultsFile, name of Kubernetes Netperf csv results file.")
	}

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
	var header []string
	var networkingType string
	var clusterZoneType string

	var bm = []metricsservice.BluemixMetric{}
	for i, line := range results {
		switch i {
		case 0:
			// Host networking flag
			hn, err := strconv.ParseBool(line[1])
			if err != nil {
				networkingType = "unknown"
			} else {
				if hn {
					networkingType = "host"
				} else {
					networkingType = "calico"
				}
			}
		case 1:
			// Multizone flag
			mz, err := strconv.ParseBool(line[1])
			if err != nil {
				clusterZoneType = "unknown"
			} else {
				if mz {
					clusterZoneType = "multizone"
				} else {
					clusterZoneType = "singlezone"
				}
			}
		case 2:
			// Header Line
			header = line
		default:
			testCase := strings.Split(line[0], ".")
			t1 := strings.Join(strings.Split(testCase[0], " ")[1:], "_")
			t2 := strings.Replace(strings.TrimSpace(strings.Join(testCase[1:], "")), " ", "_", -1)
			testDetails := strings.Join([]string{t1, t2}, ".")

			for j, testResults := range line[1 : len(line)-1] {
				var testName, metricSuffix string
				val, _ := strconv.ParseFloat(string(testResults), 64)

				if strings.Contains(header[j+1], "Maximum") {
					testName = string(testDetails)
					metricSuffix = "max"
				} else {
					if strings.Contains(testDetails, "TCP") {
						testName = strings.Join([]string{string(testDetails), strings.Replace("MSS"+header[j+1], " ", "_", -1)}, ".")
					} else {
						testName = string(testDetails)
					}
					metricSuffix = "sparse-avg"
				}

				bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{"k8s-netperf", testName, networkingType, clusterZoneType, metricSuffix}, "."), Timestamp: time.Now().Unix(), Value: val})
			}
		}
	}
	if verbose {
		for _, m := range bm {
			fmt.Println(m)
		}
	}

	if metrics {
		metricsservice.WriteBluemixMetrics(bm, true, "", "")
	}
}
