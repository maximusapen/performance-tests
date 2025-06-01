/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	bm "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"

	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	client "github.com/influxdata/influxdb1-client/v2"
)

var (
	verbose    bool
	metricsDir string
	testName   string
	dbKey      string
	bmValue    string
)

// Take the output files from a set of parallel test runs and combine
// them into a single set of metrics to be uploaded to Influx.
func writeFilesToInfluxdb() {

	var fileName string
	var metricsCfg bm.ServiceConfig
	ok := true

	if metricsCfg, ok = bm.ReadMetricsTomlFile(); !ok {
		return
	}

	httpClient, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     "http://" + metricsCfg.Metrics.InfluxdbHost + ":" + metricsCfg.Metrics.InfluxdbPort,
		Username: metricsCfg.Metrics.InfluxdbUser,
		Password: dbKey, // pragma: allowlist secret
		Timeout:  300 * time.Second,
	})
	if err != nil {
		log.Printf("Failed to create http client to Influxdb. No metric data will not be sent.\nError %s\n" + err.Error())
		return
	}

	defer httpClient.Close()

	var fileData []bm.InfluxMetricArray
	fileNum := 0
	for {
		fileNum++
		fileName = metricsDir + "/" + testName + strconv.Itoa(fileNum) + ".json"

		_, err := os.Stat(fileName)
		// Continue processing incremental filenames (e.g. sysbench0.json, sysbench1.json ...)
		// until we run out of files.
		if err != nil {
			// Don't print the error if the file is missing ... this is the expected exit point.
			// Any other error is unexpected so we will stop processing and report the issue.
			if !os.IsNotExist(err) {
				log.Printf("Error checking file stats: %s\n %s\n", fileName, err.Error())
				return
			}
			fileNum-- // Previous number file was last successful
			log.Printf("Processing of %v files complete.\n", fileNum)
			break

		} else {

			log.Printf("Processing file: %s\n", fileName)
			// #nosec G304
			file, err := ioutil.ReadFile(fileName)
			if err != nil {
				log.Printf("Error opening metrics data file: %s\n %s\n", fileName, err.Error())
				return
			}

			data := bm.InfluxMetricArray{}

			// Read json from file
			err = json.Unmarshal([]byte(file), &data)
			if err != nil {
				log.Printf("Error reading metrics data from file: %s\n %s\n", fileName, err.Error())
			}

			// Append to our data array
			fileData = append(fileData, data)
		}
	}

	// Now we have all of the file data in memory we can loop around
	// it to combine the values for the parallel run.
	metricTotals := make(map[string]float64)
	for _, file := range fileData {
		for _, metric := range file.InfluxMetricArray {
			for key, value := range metric.Fields {

				// As the value in Fields is an interface{} we need to do
				// a cast to float64 here so that we can do our arithmatic.
				floatValue, ok := value.(float64)
				if !ok {
					// Handle error case
					log.Printf("JSON metric value is not a float64: %v", value)
					return
				}

				// NOTE: We rely on the zero value of an entry in our
				//       totals map being zero here the first time we
				//       see a metric.
				metricTotals[key] += float64(floatValue)
			}
		}
	}

	// Create our final combined file values using the data from the
	// first actual file to fill in all of the non-metric value data.
	finalData := bm.InfluxMetricArray{}
	metricNum := 0
	for key, total := range metricTotals {
		// Create our new Fields value using the total
		// of all parallel runs combined.
		fields := map[string]interface{}{
			key: total,
		}

		// Append this metric to our array using the new
		// average metric value stored in fields.
		finalData.InfluxMetricArray = append(finalData.InfluxMetricArray,
			bm.InfluxDataStruc{
				DBTableName: fileData[0].InfluxMetricArray[metricNum].DBTableName,
				Tags:        fileData[0].InfluxMetricArray[metricNum].Tags,
				Fields:      fields,
				TimeStamp:   fileData[0].InfluxMetricArray[metricNum].TimeStamp,
			})

		// Increment the metric counter
		metricNum++
	}

	// Initialise our Influx batch point
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  metricsCfg.Metrics.InfluxdbName,
		Precision: "ms",
	})
	if err != nil {
		log.Printf("Failed to create Influxdb batch point.\nError %s\n" + err.Error())
		return
	}

	// Add each metric array element to the Ingress data structure ready for a batch write to Influxdb
	for _, p := range finalData.InfluxMetricArray {
		// Check for override of carrier name. (Required when running against non stage performance carriers)
		if mro := os.Getenv("METRICS_ROOT_OVERRIDE"); len(mro) > 0 {
			p.Tags["CarrierName"] = mro
		}

		// For Satellite clusters this is nasty, but until we come up with a better and consistent standard for naming carrier/sateliite envs....
		if ml := os.Getenv("METRICS_LOCATION"); len(ml) > 0 {
			cn := p.Tags["CarrierName"]
			cn = strings.Split(cn, "_")[0]                      // Get the carrier name from metrics config file
			cn = strings.Replace(cn, "carrier", "satellite", 1) // Replace carrier with satellite
			cn = strings.Join([]string{cn, ml}, "-")            // Finally add on the location name
			p.Tags["CarrierName"] = cn
		}

		if verbose {
			fmt.Println(p.DBTableName, p.Tags, p.Fields, p.TimeStamp)
		}
		pt, err := client.NewPoint(p.DBTableName, p.Tags, p.Fields, p.TimeStamp)
		if err != nil {
			fmt.Println("Error creating Influxdb data point: ", err.Error())
		} else {
			bp.AddPoint(pt)
		}
	}

	// write the batch of data to Influxdb
	if err := httpClient.Write(bp); err != nil {
		log.Printf("Failed to send request to Influxdb.\nError: %s\n ", err.Error())
	} else {
		log.Println("Metrics successfully sent to influxdb")
	}
	if err := httpClient.Close(); err != nil {
		log.Printf("Error closing Influxdb client.\nError: %s\n ", err.Error())
	}
}

// Main - sends the supplied value to the metrics service
func main() {
	log.SetOutput(os.Stdout)

	flag.StringVar(&metricsDir, "metricsdir", "/performance/metrics", "Directory containing metrics results files")
	flag.StringVar(&testName, "testname", "", "The name of the test. Metrics files will have been created in the form testName0.json. e.g. sysbench0.json, sysbench1.json ...")
	flag.StringVar(&dbKey, "dbkey", "", "Ingress db password")
	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.Parse()

	flagset := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { flagset[f.Name] = true })

	if !flagset["testname"] {
		fmt.Printf("\n\n**Error** : testname flag need to be set. \n")
		return
	}
	if len(dbKey) == 0 {
		dbKey = os.Getenv("METRICS_DB_KEY")
	}

	writeFilesToInfluxdb()
}
