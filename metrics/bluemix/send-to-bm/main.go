/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2019, 2020 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
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
	verbose         bool
	testName        string
	dbKey           string
	metricsTestName string
	bmValue         string
)

// Take the output of the CSV file and send to the metrics service
func writeSingleMetric() {

	if bmValueFloat, err := strconv.ParseFloat(bmValue, 64); err == nil {

		var bm = []metricsservice.BluemixMetric{}

		// populate the metrics
		bm = append(bm, metricsservice.BluemixMetric{Name: strings.Join([]string{metricsTestName, "sparse-avg"}, "."), Timestamp: time.Now().Unix(), Value: bmValueFloat})

		if verbose {
			for _, m := range bm {
				fmt.Println(m)
			}
		}

		metricsservice.WriteBluemixMetrics(bm, true, testName, dbKey)

	} else {
		fmt.Printf("\n\n**Error** : Unable to convert bmval string value to float. No metrics will be sent.\n Error: %s", err)
		return
	}

}

// Main - sends the supplied value to the metrics service
func main() {
	log.SetOutput(os.Stdout)

	flag.StringVar(&bmValue, "bmval", "", "Metric value to be written to Bluemix")
	flag.StringVar(&testName, "testname", "", "Test name in Jenkins - only needed if sending alerts to RazeeDash")
	flag.StringVar(&dbKey, "dbkey", "", "Metrics database key - only needed if sending metrics to database")
	flag.StringVar(&metricsTestName, "metricsTestName", "", "Specific name to be used in metrics path i.e HTTPNodePort.nodes5_replica3 rather than the overall test name - httperf")
	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.Parse()

	flagset := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { flagset[f.Name] = true })

	if !flagset["bmval"] {
		fmt.Printf("\n\n**Error** : bmval flag not set. It should be the value to send to bluemix\n")
		return
	}

	if len(dbKey) == 0 {
		dbKey = os.Getenv("METRICS_DB_KEY")
	}

	writeSingleMetric()
}
