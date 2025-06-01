/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2021, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package config

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

const (
	// ColourReset - Normal text
	ColourReset = "\033[0m"
	// ColourBold - Bold text
	ColourBold = "\033[1m"
	// ColourFaint - Faint text
	ColourFaint = "\033[2m"
	// ColourUnderline - Underlined text
	ColourUnderline = "\033[4m"
	// ColourRed - Red text
	ColourRed = "\033[31m"
	// ColourGreen - Greem text
	ColourGreen = "\033[32m"
	// ColourYellow - Yellow text
	ColourYellow = "\033[33m"
	// ColourBlue - Blue text
	ColourBlue = "\033[34m"
	// ColourMagenta - Magenta text
	ColourMagenta = "\033[35m"
)

// Weekday supports yaml parsing of sring representation of days of week
type Weekday struct {
	Weekday time.Weekday
}

// UnmarshalYAML supports yaml parsing of sring representation of days of week
func (w *Weekday) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var buf string
	err := unmarshal(&buf)
	if err != nil {
		return nil
	}
	daysOfWeek := map[string]time.Weekday{
		"Sunday":    time.Sunday,
		"Monday":    time.Monday,
		"Tuesday":   time.Tuesday,
		"Wednesday": time.Wednesday,
		"Thursday":  time.Thursday,
		"Friday":    time.Friday,
		"Saturday":  time.Saturday,
	}
	dw, ok := daysOfWeek[strings.TrimSpace(buf)]
	if !ok {
		return errors.New("invalid weekday")
	}

	w.Weekday = dw
	return nil
}

// Contains checks if the weekday is contained within the Weekday slice
func Contains(s []Weekday, e time.Weekday) bool {
	for _, w := range s {
		if w.Weekday == e {
			return true
		}
	}

	return false
}

// ConfigData holds alert configuration information
var ConfigData Data

const (
	// Never : Never send Slack DM summary
	Never SlackNotification = iota
	//WhenFound : Only send Slack DM summary when alerts are found
	WhenFound
	//Always : Always send Slack DM summary, whether alerts found or not
	Always
)

// SlackNotification is an enumeration of available Slack DM notification options
type SlackNotification int

// UnmarshalYAML supports yaml parsing of sring representation of notification types
func (w *SlackNotification) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var buf string
	err := unmarshal(&buf)
	if err != nil {
		return nil
	}
	nt := map[string]SlackNotification{
		"Never":     Never,
		"WhenFound": WhenFound,
		"Always":    Always,
	}
	n, ok := nt[strings.TrimSpace(buf)]
	if !ok {
		return errors.New("invalid notification type")
	}

	*w = n
	return nil
}

// Data holds top level configuration information
type Data struct {
	InfluxDB     InfluxDB
	Slack        Slack
	Options      Options
	Environments TestEnvironments
	Tests        []Test
}

// TestEnvironments holds test environment configuration data keyed on environment name
type TestEnvironments map[string]TestEnvironment

// TestEnvironment holds test environment configuration data for an environment
type TestEnvironment struct {
	Carrier         string
	MachineType     []string `yaml:"machineType"`
	KubeVersion     []string `yaml:"kubeVersion"`
	OperatingSystem []string `yaml:"operatingSystem"`
	Owner           Owner
}

// Owner holds information about a test environment owner
type Owner struct {
	Name   string
	Slack  string
	Notify SlackNotification
	Days   []Weekday
}

// InfluxDB holds configuration data for connecting to an influx db instance
type InfluxDB struct {
	Host     string
	Port     int
	Database string
	Username string
	Timeout  int
}

// Slack holds configuration data for reporting alerts via Slack
type Slack struct {
	Enabled    bool
	Channel    string
	ResultsURL string `yaml:"resultsURL"`
}

// History holds configuration data for determing the amount and age of historical results data
type History struct {
	Count   int // Maximum number of test data results to process
	Days    int // Maximum age of test data to process
	Current int // Number of results to be considered as the latest test result (value used is the mean of these results)
	Minimum int // Minimum number of historical results for generating z-score / informational alerts
}

// Options holds high level configuration options
type Options struct {
	History  History
	Debug    bool
	Verbose  bool
	Failures bool
	Leniency float64 // Z-Score comparing an alert threshold against actual results. Thresholds which exceed results by this value should be considered for becoming more aggressive.
}

// Test is an individual performance test
type Test struct {
	Name        string
	Environment []AlertEnvironment
	DisableInfo bool `yaml:"disableInfo"`
}

// AlertEnvironment defines the set of alerts for a set of Kubernetes version
// and operating system combinations.
type AlertEnvironment struct {
	KubeVersion     []string `yaml:"kubeVersion"`
	OperatingSystem []string `yaml:"operatingSystem"`
	Alerts          []Alert
}

// Thresholds defines the simple values for generating the various alert severities.
type Thresholds struct {
	Warn   float64
	Error  float64
	Zscore float64
}

// Issue holds a Github issue URL for an alert and its associated Kube version(s)
// and operating systems
type Issue struct {
	KubeVersion     []string `yaml:"kubeVersion"`
	OperatingSystem []string `yaml:"operatingSystem"`
	Issue           string
}

// Alert defines a single alert
type Alert struct {
	Name       string
	LimitType  string `yaml:"limitType"`
	Issues     map[string]Issue
	Thresholds map[string]Thresholds
}

// getConfigPath returns the path of the yaml configuration config file
func getConfigPath() string {
	configPath := os.Getenv("PERF_ALERTS_CONFIG_PATH")
	if configPath != "" {
		return configPath
	}
	goPath := os.Getenv("GOPATH")
	perfSrcPath := filepath.Join("src", "github.ibm.com", "alchemy-containers", "armada-performance", "metrics", "alerting")
	return filepath.Join(goPath, perfSrcPath, "config")
}

// GetConfig returns the alerting configuration information
func GetConfig() *Data {
	if len(ConfigData.Tests) == 0 {
		// Read the contents of the alerting configuration yaml file
		configFile := filepath.Join(getConfigPath(), "perf-alerts.yaml")
		cfc, err := ioutil.ReadFile(configFile)
		if err != nil {
			log.Fatalf("Error reading alert configuration file: %s", err.Error())
		}

		// Parse the yaml configuration data
		err = yaml.Unmarshal(cfc, &ConfigData)
		if err != nil {
			log.Fatalf("Error parsing alert configuration file: %s", err.Error())
		}
	}

	return &ConfigData
}
