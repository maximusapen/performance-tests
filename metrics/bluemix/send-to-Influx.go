/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2019, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package metricsservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	client "github.com/influxdata/influxdb1-client/v2"
)

var writeToFile = false
var firstTime = true
var carrierMetricFields = []string{"cpu_pcnt_used", "memory_pcnt_used", "eth0_network_receive_private", "eth0_network_transmit_private", "eth1_network_receive_public", "eth1_network_transmit_public", "xvda_disk_pcnt_busy", "xvdb_disk_pcnt_busy", "xvdc_disk_pcnt_busy"}

// InfluxMetricArray is the influx structure
type InfluxMetricArray struct {
	InfluxMetricArray []InfluxDataStruc `json:"influxMetricArray"`
}

// InfluxDataStruc is the influx structure
type InfluxDataStruc struct {
	DBTableName string                 `json:"DBTableName"`
	Tags        map[string]string      `json:"Tags"`
	Fields      map[string]interface{} `json:"Fields"`
	TimeStamp   time.Time              `json:"TimeStamp"`
}

func metricNameContains(aMetricName string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(aMetricName, sub) {
			return false
		}
	}
	return true
}

// WriteInfluxdbData obtain tag metadata and send the data to Influxdb.
func WriteInfluxdbData(metrics []BluemixMetric, testName string, metricsPrefix string, metricsRootName string, k8sVersionShort string, host string, port string, dbName string, user string, pw string, verbose bool, clusterNames []string, carrierMetrics bool) {

	var influxMetricArray InfluxMetricArray
	var cruiserMetrics bool
	var fullFileName string
	var fileErr error
	var bp client.BatchPoints
	var httpClient client.Client
	var err error
	filePath := "/performance/metrics/"

	if len(testName) == 0 {
		log.Println("No test name specified. No metrics data will be sent to Influxdb.")
		return
	}

	if firstTime || strings.HasPrefix(testName, "dummycruisermaster") || strings.HasPrefix(testName, "cruiserchurn") {
		// Check Influxdb connectivity - if we can't connect we will write the metrics to a file
		// For long running tests (cruiser_mon & cruiserchurn check everytime, otherwise intermiitent issues can cause results
		// to be writtento file forever)
		firstTime = false
		conn, err := net.DialTimeout("tcp", host+":"+port, time.Duration(5)*time.Second)
		if conn != nil {
			defer conn.Close()
			log.Printf("Connection to Influxdb succeeded\n")
			writeToFile = false
		} else {
			fmt.Printf("No connection available to Influxdb (expected if metrics are sent from a Cruiser), so writing metrics to a local file. Reason for failure: %s\n", err)
			writeToFile = true
		}
	}

	if len(pw) == 0 {
		log.Println("No DB password specified, so writing metrics to a local file. ")
		writeToFile = true
	}

	if writeToFile {
		fileName := testName

		if _, dirErr := os.Stat(filePath); os.IsNotExist(dirErr) {
			err := os.MkdirAll(filePath, 0664)
			if err != nil {
				log.Printf("Error creating Influxdb file directory: %s\n%s\n", filePath, err)
				return
			}
		}

		// Find the next unused incremental filename e.g. sysbench1.json, sysbench2.json ...
		fileNum := 0
		for {
			fileNum++
			fullFileName = filePath + fileName + strconv.Itoa(fileNum) + ".json"
			if _, fileErr = os.Stat(fullFileName); fileErr != nil {
				break
			}
		}

		// We are looking for an unused filename, so IsNotExist should be true. For any other error, stop processing and report the cause.
		if !os.IsNotExist(fileErr) {
			log.Printf("Unexpected error when looking for Influxdb metrics filename. Last metrics name checked was:\nError %s\n" + fileErr.Error())
			return
		}
		log.Printf("Writing metrics to file: %s\n", fullFileName)

	} else {
		// Write directly to Influxdb

		httpClient, err = client.NewHTTPClient(client.HTTPConfig{
			Addr:     "http://" + host + ":" + port,
			Username: user,
			Password: pw, // pragma: allowlist secret
			Timeout:  300 * time.Second,
		})
		if err != nil {
			log.Printf("Failed to create http client to Influxdb. No metric data will not be sent.\nError %s\n" + err.Error())
			return
		}
		defer httpClient.Close()

		bp, err = client.NewBatchPoints(client.BatchPointsConfig{
			Database:  dbName,
			Precision: "ms",
		})

		if err != nil {
			log.Printf("Failed to create Influxdb batch point.\nError %s\n" + err.Error())
			return
		}

	}

	var carrierName string
	// Allow override of metrics root specified in metrics.toml file
	if mro := os.Getenv("METRICS_ROOT_OVERRIDE"); len(mro) == 0 {
		carrierName = "carrier_stage"
		rootMetricNameParts := strings.Split(metricsRootName, ".")
		for _, metricNameSubstring := range rootMetricNameParts {
			if strings.Contains(metricNameSubstring, "carrier") || strings.Contains(metricNameSubstring, "satellite") {
				carrierName = metricNameSubstring
			}
		}
	} else {
		carrierName = mro
	}

	// For Satellite clusters (but we'll generalize the suport) we need to support the identification of control plane configuration.
	if ml := os.Getenv("METRICS_LOCATION"); len(ml) > 0 {
		carrierName = strings.Join([]string{carrierName, ml}, "-")
	}

	operatingSystem := os.Getenv("METRICS_OS")

	tags := map[string]string{
		"CarrierName":     carrierName,
		"MachineType":     metricsPrefix,
		"KubeVersion":     k8sVersionShort,
		"OperatingSystem": operatingSystem,
		"TestName":        testName,
	}

	for _, ametric := range metrics {

		metricFloatValue, err := interfaceToFloat64(ametric.Value)
		if err != nil {
			log.Printf("send-to-Influx.WriteToInfluxdb: \n" + err.Error())
			return
		}

		metricName := ametric.Name
		//If the clusterName is specified then we have been called from createCluster and we need to remove the cluster name from the metrics (because we can't use wild cards in Influxdb metric value fields)
		clusterNameTag := ""
		if len(clusterNames) > 0 {
			for _, clusterName := range clusterNames {
				if strings.Contains(metricName, clusterName) {
					clusterNameTag = clusterName
					metricName = strings.Replace(metricName, (clusterName + "."), "", 1)
				}

			}
		}

		//replace all chars that will not be valid in influxdb .. and remove the word sparse-avg
		badCharReplacer := strings.NewReplacer(".sparse-avg", "", "-", "_", ".", "_", "(", "_", ")", "", " ", "_")
		testNameGoodChars := badCharReplacer.Replace(testName)
		metricNameGoodChars := badCharReplacer.Replace(metricName)
		//remove the testname from the beginning of the metrics name just to shorten the name - first occurrence only
		shortMetricName := strings.Replace(metricNameGoodChars, (testNameGoodChars + "_"), "", 1)
		fieldName := shortMetricName

		dbTableName := testNameGoodChars
		cruiserMetrics = false
		metricType := "custom"

		if carrierMetrics {

			for _, metricField := range carrierMetricFields {
				if metricNameContains(shortMetricName, metricField) {
					metricType = metricField
					fieldName = metricField
				}
			}

		} else {
			cruiserMetrics = true

			if metricNameContains(shortMetricName, "cruiser_namespace_metrics_", "_cpu") {
				metricType = "ns_cpu"
			} else if metricNameContains(shortMetricName, "cruiser_namespace_metrics_", "_mem") {
				metricType = "ns_mem"
			} else if metricNameContains(shortMetricName, "cruiser_node_metrics_", "_cpu") {
				metricType = "node_cpu"
			} else if metricNameContains(shortMetricName, "cruiser_node_metrics_", "_mem") {
				metricType = "node_mem"
			} else if metricNameContains(shortMetricName, "cruiser_pod_metrics_", "_cpu") {
				metricType = "pod_cpu"
			} else if metricNameContains(shortMetricName, "cruiser_pod_metrics_", "_mem") {
				metricType = "pod_mem"
			} else {
				cruiserMetrics = false
			}

			if cruiserMetrics {
				//remove the following cruiser metric strings to improve the grafana presentation of the names
				cruiserMetricNameReplacer := strings.NewReplacer("cruiser_namespace_metrics_", "", "cruiser_node_metrics_", "", "cruiser_pod_metrics_", "")
				shortMetricName = cruiserMetricNameReplacer.Replace(shortMetricName)
				fieldName = metricType
			}
		}

		tags["MetricName"] = shortMetricName
		tags["MetricType"] = metricType
		tags["ClusterName"] = clusterNameTag

		fields := map[string]interface{}{
			fieldName: metricFloatValue,
		}

		//Only print out custom metrics - exclude metrics-server data as they generate too many log entries
		if verbose && !cruiserMetrics && !carrierMetrics {
			log.Printf("DBTable=%s  CarrierName=%s  MachineType=%s  KubeVersion=%s  OperatingSystem=%s MetricName=%s  MetricValue=%v\n", dbTableName, carrierName, metricsPrefix, k8sVersionShort, operatingSystem, shortMetricName, metricFloatValue)
		}

		if writeToFile {
			influxMetricArray.InfluxMetricArray = append(influxMetricArray.InfluxMetricArray, InfluxDataStruc{dbTableName, tags, fields, time.Now()})
		} else {
			tstamp := time.Now()
			if ametric.Timestamp > 0 {
				tstamp = time.Unix(ametric.Timestamp, 0)
			}
			pt, err := client.NewPoint(dbTableName, tags, fields, tstamp)

			if err != nil {
				fmt.Println("Error creating Influxdb data point: ", err.Error())
			} else {
				bp.AddPoint(pt)
			}
		}

	}

	if writeToFile {
		if !cruiserMetrics && !carrierMetrics { // Don't write cruiser or carrier metrics to files or we could end up writing large amounts of data. They should have direct access to the Influxdb anyway.
			metricsFile, err := json.MarshalIndent(influxMetricArray, "", " ")
			if err != nil {
				log.Printf("Error marshalling metrics Data: %s\n", err.Error())
			}
			f, err := os.OpenFile(fullFileName, os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				log.Printf("Error creating or opening file: %s.\n%s\n", fullFileName, err.Error())
				return
			}
			defer f.Close()
			if _, err = f.Write(metricsFile); err != nil {
				log.Printf("Error writing metrics data to file:\n %s\n", err.Error())
			}
		}
	} else {
		if err := httpClient.Write(bp); err != nil {
			log.Printf("Failed to send request to Influxdb.\nError: %s\n ", err.Error())
		} else {
			if verbose && !cruiserMetrics && !carrierMetrics {
				log.Println("Metrics successfully sent to influxdb")
			}

		}

		if err := httpClient.Close(); err != nil {
			log.Printf("Error closing Influxdb client.\nError: %s\n ", err.Error())
		}
	}

}

func interfaceToFloat64(interfaceValue interface{}) (float64, error) {

	var floatType = reflect.TypeOf(float64(0))
	var stringType = reflect.TypeOf("")

	switch aFloatType := interfaceValue.(type) {
	case float64:
		return aFloatType, nil
	case float32:
		return float64(aFloatType), nil
	case int:
		return float64(aFloatType), nil
	case int32:
		return float64(aFloatType), nil
	case int64:
		return float64(aFloatType), nil
	case uint:
		return float64(aFloatType), nil
	case uint32:
		return float64(aFloatType), nil
	case uint64:
		return float64(aFloatType), nil
	case string:
		strFloat, err := strconv.ParseFloat(aFloatType, 64)
		if err != nil {
			return -999, errors.New("conversion of metric string value to float 64 failed")
		}
		return strFloat, nil
	default:
		v := reflect.ValueOf(interfaceValue)
		v = reflect.Indirect(v)
		if v.Type().ConvertibleTo(floatType) {
			fv := v.Convert(floatType)
			return fv.Float(), nil
		} else if v.Type().ConvertibleTo(stringType) {
			sv := v.Convert(stringType)
			s := sv.String()
			return strconv.ParseFloat(s, 64)
		} else {
			log.Printf("Failed to parse metrics float value: %v\n", interfaceValue)
			return -999, errors.New("no conversion of this metric type in interfaceToFloat64")
		}
	}
}
