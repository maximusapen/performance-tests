/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2021, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
	"flag"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.ibm.com/alchemy-containers/armada-performance/metrics/alerting/alert"
	"github.ibm.com/alchemy-containers/armada-performance/metrics/alerting/config"
	influxdata "github.ibm.com/alchemy-containers/armada-performance/metrics/alerting/influx"
	"github.ibm.com/alchemy-containers/armada-performance/metrics/alerting/jenkins"
	"github.ibm.com/alchemy-containers/armada-performance/metrics/alerting/slack"
)

func outputEnvHeader(kv, mt, os string) {
	header := ""
	if len(kv) > 0 {
		header = fmt.Sprintf("%sKube Version: %s, ", header, kv)
	}

	if len(mt) > 0 {
		header = fmt.Sprintf("%sMachineType: %s, ", header, mt)
	}

	if len(os) > 0 {
		header = fmt.Sprintf("%sOS: %s", header, os)
	}

	if len(header) > 0 {
		fmt.Printf("\t%s\n", header)
	}
}

func main() {
	alerts := make(map[string][]alert.Alert)

	// Process the configuration file
	conf := config.GetConfig()

	// Allow command line override of some configuration options
	flag.BoolVar(&conf.Options.Verbose, "verbose", conf.Options.Verbose, "Detailed logging output")
	flag.BoolVar(&conf.Options.Debug, "debug", conf.Options.Debug, "Debug output")
	flag.BoolVar(&conf.Slack.Enabled, "slack", conf.Slack.Enabled, "Slack notifications")
	flag.Parse()

	// Ensure verbose logging if debug option specified
	if conf.Options.Debug {
		conf.Options.Verbose = true
	}

	// Data is stored in an influx database
	ic := influxdata.NewInfluxClient(conf.InfluxDB)

	// Let's sort the map to get a consistent ordering in the output
	sortedEnvs := make([]string, len(conf.Environments))
	i := 0
	for n := range conf.Environments {
		sortedEnvs[i] = n
		i++
	}
	sort.Strings(sortedEnvs)

	// For each environment (carrier)
	for _, name := range sortedEnvs {
		e := conf.Environments[name]
		//for name, e := range conf.Environments {
		if conf.Options.Verbose {
			fmt.Printf("%s%s%s%s\n", config.ColourBold, config.ColourUnderline, strings.ToUpper(name), config.ColourReset)
		}

		// For each test in the configuration
		for _, t := range conf.Tests {
			testHeaderWritten := false

			// For each machine type in the current test environment
			for _, mt := range e.MachineType {
				for _, ae := range t.Environment {
					expectResults := false

					for _, a := range ae.Alerts {
						if _, ok := a.Thresholds[mt]; ok {
							expectResults = true
							break
						}
					}

					if !expectResults {
						// Nothing to do here, move onto the next test
						continue
					}

					// For each kube version supported by the environment
					for _, kv := range e.KubeVersion {
						// For each OS supported by the environment
						for _, eos := range e.OperatingSystem {
							// For each OS supported by the alert environment
							for _, os := range ae.OperatingSystem {
								// Is the OS for this alert supported by the current environment?
								if os == eos {
								alerts:
									// For each alert defined in the alert environment
									for _, a := range ae.Alerts {
										// Do we have any theresholds defined for this alert?
										if _, ok := a.Thresholds[mt]; ok {
											kvMatch := false
											for _, tkv := range ae.KubeVersion {
												if tkv == kv {
													kvMatch = true
												}
											}

											if kvMatch {
												if conf.Options.Verbose && !testHeaderWritten {
													fmt.Printf("%s%s%s\n", config.ColourBold, t.Name, config.ColourReset)
													testHeaderWritten = true
												}

												// Silence any alerts which are known and have an active issue open
												if i, ok := a.Issues[mt]; ok {
													for _, ikv := range i.KubeVersion {
														for _, ios := range i.OperatingSystem {
															if ikv == kv && ios == os {
																if conf.Options.Verbose {
																	fmt.Printf("%s", config.ColourBlue)
																	outputEnvHeader(kv, mt, os)
																	fmt.Printf("\t\t%s: SILENCING\n", a.Name)
																	fmt.Printf("\t\t\tIssue: %s %s\n", i.Issue, config.ColourReset)
																	fmt.Println()

																	alerts[name] = append(alerts[name],
																		alert.Alert{
																			Name:            t.Name,
																			EnvName:         name,
																			Carrier:         e.Carrier,
																			Owner:           e.Owner,
																			KubeVersion:     kv,
																			MachineType:     mt,
																			OperatingSystem: os,
																			Sev:             alert.Silenced,
																		},
																	)
																	continue alerts
																}
															}
														}
													}
												}

												// Get the data from influx
												results := ic.GetTestData(e.Carrier, t.Name, kv, mt, os, a.Name, conf.Options.History)

												if len(results.Current) == 0 {
													// No results for this test/result/kube-version combination
													if conf.Options.Verbose {
														outputEnvHeader(kv, mt, os)
														fmt.Printf("\t\t%s: %s*** NO DATA ***%s\n", a.Name, config.ColourBold, config.ColourReset)
													}
													continue
												}

												if conf.Options.Verbose {
													outputEnvHeader(kv, mt, os)
												}

												if conf.Options.Verbose {
													crMin := math.MaxFloat64
													crMax := 0.0
													for _, cr := range results.Current {
														crMin = math.Min(crMin, cr.Val)
														crMax = math.Max(crMax, cr.Val)
													}
													switch a.LimitType {
													case "floor":
														fmt.Printf("\t\t%s: %.6g\n", a.Name, crMax)
													case "ceiling":
														fmt.Printf("\t\t%s: %.6g\n", a.Name, crMin)
													}
												}

												if conf.Options.Debug {
													fmt.Print(config.ColourFaint)
													for _, r := range results.Current {
														fmt.Printf("\t\t\t%s : %.6g\n", time.Unix(r.Timestamp, 0), r.Val)
													}
													for _, r := range results.Historical {
														fmt.Printf("\t\t\t%s : %.6g\n", time.Unix(r.Timestamp, 0), r.Val)
													}
													fmt.Print(config.ColourReset)
												}

												ta := alert.Alert{
													Name:              t.Name,
													EnvName:           name,
													Carrier:           e.Carrier,
													Owner:             e.Owner,
													KubeVersion:       kv,
													MachineType:       mt,
													OperatingSystem:   os,
													LeniencyThreshold: conf.Options.Leniency,
												}

												// We don't want information alerts, so update threshold
												if t.DisableInfo {
													ta.LeniencyThreshold = math.MaxFloat64
												}

												alerts[name] = append(alerts[name], ta.ProcessData(a, results)...)
											} else {
												expectResults = false
											}
										}
									} // alert env alerts
								} // os == eos
							} // alert env OS
						} // env OS
					} // env kube versions
				}
			}
		}

		if conf.Options.Verbose {
			fmt.Println("---")
			fmt.Println()
		}
	}

	// Send alerts to Slack
	if conf.Slack.Enabled {
		failures := jenkins.Failures(conf)
		slack.SendAlerts(conf, failures, alerts)
	}
}
