/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2017, 2022  All Rights Reserved.
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
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	metricsservice "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"
	apiresponsiveness "github.ibm.com/alchemy-containers/armada-performance/metrics/kubernetes-e2e/apiResponsiveness"
	apiresponsivenessoverall "github.ibm.com/alchemy-containers/armada-performance/metrics/kubernetes-e2e/apiResponsivenessOverall"
	"github.ibm.com/alchemy-containers/armada-performance/metrics/kubernetes-e2e/config"
	e2emetrics "github.ibm.com/alchemy-containers/armada-performance/metrics/kubernetes-e2e/e2eMetrics"
)

// Holds our metrics to be sent to IBM Cloud (Bluemix) metric service
var bm []metricsservice.BluemixMetric
var nodeCountStr string

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

// Function to check whether the value equals the zero value for its type or is empty
func isZeroOrEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Func:
		return v.IsNil()
	case reflect.Map, reflect.Slice:
		return v.IsNil() || v.Len() == 0
	case reflect.Array:
		z := true
		for i := 0; i < v.Len(); i++ {
			z = z && isZeroOrEmpty(v.Index(i))
		}
		return z
	case reflect.Struct:
		z := true
		for i := 0; i < v.NumField(); i++ {
			z = z && isZeroOrEmpty(v.Field(i))
		}
		return z
	case reflect.Invalid:
		return true
	}
	// Compare other types directly:
	z := reflect.Zero(v.Type())
	return v.Interface() == z.Interface()
}

func processE2E(conf config.MetricsForE2E, testcase string, bytes *[]byte) {
	var mfe2eContents e2emetrics.MetricsForE2ETests
	json.Unmarshal(*bytes, &mfe2eContents)
	for _, data := range mfe2eContents.APIServerMetrics.APIServerRequestLatenciesSummary {
		if val, ok := conf.Latency[data.Metric.Resource]; ok {
			for _, v := range val {
				if v.Verb == data.Metric.Verb && v.Quantile == data.Metric.Quantile && v.SubResource == data.Metric.SubResource {
					var resourceStr = data.Metric.Resource
					if len(data.Metric.SubResource) > 0 {
						resourceStr = strings.Join([]string{resourceStr, data.Metric.SubResource}, "-")
					}
					quantileVal, err := strconv.ParseFloat(data.Metric.Quantile, 32)
					if err != nil {
						log.Printf("Results quantile data \"%s\"is incorrectly formatted. Ignoring metric.\n", data.Metric.Quantile)
					} else {
						dataVal, ok := data.Value[1].(string)
						if ok {
							quantileName := "Perc" + strconv.FormatFloat(quantileVal*100.0, 'f', -1, 32)
							if config.Verbose {
								fmt.Printf("%s:\"", data.Metric.Name)
								fmt.Printf("%s %s\"\n", data.Metric.Verb, resourceStr)
								fmt.Printf("%s: %s\n", quantileName, dataVal)
							}

							if mv, err := strconv.Atoi(dataVal); err == nil {
								if config.IBMMetrics {
									bm = append(bm,
										metricsservice.BluemixMetric{
											Name:      strings.Join([]string{testcase, data.Metric.Name, nodeCountStr, data.Metric.Verb, resourceStr, quantileName, "max"}, "."),
											Timestamp: time.Now().Unix(),
											Value:     mv,
										},
									)
								}
							} else {
								log.Printf("MetricsForE2E results data \"%s\" is not in expected format. Ignoring non numeric metric.\n", data.Metric.Name)
							}
						} else {
							log.Printf("MetricsForE2E results data \"%s\" is not in expected format. Ignoring metric.\n", data.Metric.Name)
						}
					}
				}
			}
		}
	}
}

func processAPIResponsiveness(conf map[string][]config.APIResponsiveness, testcase string, bytes *[]byte) {
	var apiResponseContents apiresponsiveness.Latency
	json.Unmarshal(*bytes, &apiResponseContents)
	for _, data := range apiResponseContents.DataItems {
		var dataLabel, tcLabel string

		// APIResponsiveness
		subTestcase := strings.Split(testcase, ".")[1]
		switch subTestcase {
		case "APIResponsiveness":
			resourceStr := data.Labels["Resource"]
			scope := data.Labels["Scope"]
			dataLabel = resourceStr
			if len(data.Labels["Subresource"]) > 0 {
				resourceStr = strings.Join([]string{resourceStr, data.Labels["Subresource"]}, "-")
			}

			tcLabel = strings.Join([]string{data.Labels["Verb"], scope, resourceStr}, ".")
		case "PodStartupLatency":
			dataLabel = data.Labels["Metric"]
			tcLabel = dataLabel
		default:
			{
				fmt.Printf("Unrecognised testcase \"%s\"", testcase)
			}
		}

		if val, ok := conf[dataLabel]; ok {
			for _, v := range val {
				if v.Verb == data.Labels["Verb"] && v.SubResource == data.Labels["Subresource"] {
					if config.Verbose {
						fmt.Printf("\"%s\"\n", strings.Replace(tcLabel, ".", " ", 1))
					}
					for _, d := range v.Data {
						if config.Verbose {
							fmt.Printf("%s: %g\n", d, data.Data[d])
						}

						if config.IBMMetrics {
							bm = append(bm,
								metricsservice.BluemixMetric{
									Name:      strings.Join([]string{testcase, nodeCountStr, tcLabel, d, "max"}, "."),
									Timestamp: time.Now().Unix(),
									Value:     data.Data[d],
								},
							)
						}
					}
				}
			}
		}
	}
}

func processAPIResponsivenessOverall(conf map[string][]config.APIResponsivenessOverall, testcase string, bytes *[]byte) {
	var apiResponseOverallContents apiresponsivenessoverall.OverallLatency
	json.Unmarshal(*bytes, &apiResponseOverallContents)
	for _, data := range apiResponseOverallContents.Data.Result {
		var dataLabel, tcLabel string
		resourceStr := data.Metric["resource"]
		subresourceStr := data.Metric["subresource"]

		scope := data.Metric["scope"]
		dataLabel = resourceStr
		if len(subresourceStr) > 0 {
			resourceStr = strings.Join([]string{resourceStr, subresourceStr}, "-")
		}
		tcLabel = strings.Join([]string{data.Metric["verb"], scope, resourceStr}, ".")

		if val, ok := conf[dataLabel]; ok {
			for _, v := range val {
				if v.Verb == data.Metric["verb"] && v.SubResource == data.Metric["subresource"] {
					if config.Verbose {
						fmt.Printf("\"%s\"\n", strings.Replace(tcLabel, ".", " ", 1))
					}

					dataVal, ok := data.Value[1].(string)
					if ok {
						if config.Verbose {
							fmt.Printf("Value: %s\n", dataVal)
						}
					}
					if dataVal != "NaN" {
						if mv, err := strconv.ParseFloat(dataVal, 5); err == nil {
							if config.IBMMetrics {
								bm = append(bm,
									metricsservice.BluemixMetric{
										Name:      strings.Join([]string{testcase, nodeCountStr, tcLabel, "mean"}, "."),
										Timestamp: time.Now().Unix(),
										Value:     mv * 1000,
									},
								)
							}
						}
					}
				}
			}
		}
	}
}

func processTestPhaseTimer(conf struct{ Report bool }, testcase string, bytes *[]byte) {
	var testPhaseTimerContents apiresponsiveness.Latency // Test phase timer follows same format as APIResponsiveness latency data
	json.Unmarshal(*bytes, &testPhaseTimerContents)

	if conf.Report {
		for _, data := range testPhaseTimerContents.DataItems {
			for k, v := range data.Data {
				if config.Verbose {
					fmt.Printf("%s : %g\n", k, v)
				}

				m := strings.Replace(k, " ", "-", -1)
				if config.IBMMetrics {
					bm = append(bm,
						metricsservice.BluemixMetric{
							Name:      strings.Join([]string{testcase, nodeCountStr, m, "max"}, "."),
							Timestamp: time.Now().Unix(),
							Value:     v,
						},
					)
				}
			}
		}
	}
}

func main() {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	flag.StringVar(&config.ResultsDir, "resultsDir", "", "dirctory containing Kubernetes E2E testing results files")
	flag.StringVar(&config.TestcasesFile, "config", "metrics.toml", "(optional) name of file describing Kubernetes E2E testing metrics to process")
	flag.BoolVar(&config.Verbose, "verbose", false, "Write data to stdout. Defaults to no verbose output")
	flag.BoolVar(&config.IBMMetrics, "metrics", false, "Send results/metrics to IBM Cloud Metrics service. Defaults to no metrics")
	flag.Parse()

	if len(config.ResultsDir) == 0 {
		// Default to current working directory
		config.ResultsDir, _ = os.Getwd()
	}

	// Check we've been given a valid directory
	if fileInfo, resultsDirErr := os.Stat(config.ResultsDir); resultsDirErr != nil {
		log.Fatalln(resultsDirErr.Error())
	} else {
		if !fileInfo.IsDir() {
			log.Fatalf("%s is not a directory\n", config.ResultsDir)
		}
	}

	// Get a list of files in our results directory
	resultsFiles, err := ioutil.ReadDir(config.ResultsDir)
	if err != nil {
		log.Fatalln(err.Error())
	}

	// Read config file to determine which Kubernetes E2E Test metrics we're interested in
	var conf config.Config
	config.ParseConfig(filepath.Join(config.GetConfigPath(), config.TestcasesFile), &conf)

	_, schedulableNodes, hollowNodes := config.GetClusterNodeCount(kubeconfig)
	var nodeStr string
	if hollowNodes {
		nodeStr = "hollow_nodes"
	} else {
		nodeStr = "nodes"
	}
	nodeCountStr = nodeStr + strconv.Itoa(schedulableNodes)

	// Process each results file in turn
	for _, rf := range resultsFiles {
		// We're only interested in json files
		if strings.HasSuffix(rf.Name(), ".json") {
			v := reflect.ValueOf(conf)
			filenameComponents := strings.Split(rf.Name(), "_")

			// This section deals with the fact that the new clusterloader2 code added an extra filename component
			// to a few of the files at the beginning of the name.
			// Those files with 4 parts to the name can happily ignore the first component.
			if len(filenameComponents) == 4 {
				filenameComponents = filenameComponents[1:]
			}

			// Basic sanity checks
			if v.Kind() == reflect.Struct && len(filenameComponents) >= 2 {
				v = v.FieldByName(strings.Title(filenameComponents[1]))
				v = v.FieldByName(strings.Title(filenameComponents[0]))

				// Check if we're interested in metrics from this file
				if !isZeroOrEmpty(v) {
					testcase := strings.Join([]string{filenameComponents[1], filenameComponents[0]}, ".")
					if config.Verbose {
						fmt.Println(testcase)
						fmt.Println(strings.Repeat("-", utf8.RuneCountInString(testcase)))
					}

					bytes, err := ioutil.ReadFile(filepath.Join(config.ResultsDir, rf.Name()))
					if err != nil {
						log.Fatalln(err.Error())
					}

					switch testcase {
					case "load.MetricsForE2E":
						{
							processE2E(conf.Load.MetricsForE2E, testcase, &bytes)
						}
					case "density.MetricsForE2E":
						{
							processE2E(conf.Density.MetricsForE2E, testcase, &bytes)
						}
					case "density.APIResponsiveness":
						{
							processAPIResponsiveness(conf.Density.APIResponsiveness, testcase, &bytes)
						}
					case "load.APIResponsivenessOverall":
						{
							processAPIResponsivenessOverall(conf.Load.APIResponsivenessOverall, testcase, &bytes)
						}
					case "density.PodStartupLatency":
						{
							processAPIResponsiveness(conf.Density.PodStartupLatency, testcase, &bytes)
						}
					case "load.PodStartupLatency":
						{
							processAPIResponsiveness(conf.Load.PodStartupLatency, testcase, &bytes)
						}
					case "load.APIResponsiveness":
						{
							processAPIResponsiveness(conf.Load.APIResponsiveness, testcase, &bytes)
						}
					case "density.SchedulingLatency":
						{
							log.Printf("\"%s\" not yet implemented. Ignoring\n", testcase)
						}
					case "load.TestPhaseTimer":
						{
							processTestPhaseTimer(conf.Load.TestPhaseTimer, testcase, &bytes)
						}
					case "density.TestPhaseTimer":
						{
							processTestPhaseTimer(conf.Density.TestPhaseTimer, testcase, &bytes)
						}
					default:
						{
							log.Printf("Unknown \"%s\". Ignoring\n", testcase)
						}
					}
					if config.Verbose {
						fmt.Println()
					}
				}
			}
		}
	}

	if config.IBMMetrics {
		if config.Verbose {
			fmt.Println(bm)
		}

		if len(bm) > 0 {
			metricsservice.WriteBluemixMetrics(bm, true, "", "")
		}
	}
}
