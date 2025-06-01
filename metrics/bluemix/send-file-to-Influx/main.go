/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2019, 2022 All Rights Reserved.
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

// Take the output of the CSV file and send to the metrics service
func writeFileToInfluxdb() {

	var fileName string
	var metricsCfg bm.ServiceConfig
	ok := true

	if metricsCfg, ok = bm.ReadMetricsTomlFile(); !ok {
		return
	}

	//metricsCfg.Metrics.InfluxdbHost, metricsCfg.Metrics.InfluxdbPort, metricsCfg.Metrics.InfluxdbName, metricsCfg.Metrics.InfluxdbUser, dbKey, metricsCfg.Metrics.InfluxdbVerbose, metricsCfg.Metrics.InfluxdbFileLimit
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  metricsCfg.Metrics.InfluxdbName,
		Precision: "ms",
	})
	if err != nil {
		log.Printf("Failed to create Influxdb batch point.\nError %s\n" + err.Error())
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

	fileNum := 0
	for {
		fileNum++
		fileName = metricsDir + "/" + testName + strconv.Itoa(fileNum) + ".json"

		_, err := os.Stat(fileName)
		// Continue processing incremental filenames (e.g. sysbench0.json, sysbench1.json ...) until we run out of files
		if err != nil {
			// Don't print the error if the file is missing ... this is the expected exit point. Any other error is unexpected so we will stop processing and report the issue.
			if !os.IsNotExist(err) {
				log.Printf("Error checking file stats:\n %s\n", err.Error())
				return
			}
			break

		} else {

			log.Printf("\n\nProcessing file: %s\n", fileName)
			// #nosec G304
			file, err := ioutil.ReadFile(fileName)
			if err != nil {
				log.Printf("Error opening metrics data file:%s\n %s\n", fileName, err.Error())
				return
			}

			data := bm.InfluxMetricArray{}

			// Read json from file
			err = json.Unmarshal([]byte(file), &data)
			if err != nil {
				log.Printf("Error reading metrics data from file:\n %s\n", err.Error())
			}

			// Add each metric array element to the Ingress data structure ready for a batch write to Influxdb
			for _, p := range data.InfluxMetricArray {
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

	writeFileToInfluxdb()

}
