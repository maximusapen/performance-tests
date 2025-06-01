/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/slack-go/slack"
	metrics "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"
)

type versionDefaults struct {
	storedDefault  string
	currentDefault string
}

// NOTE: The key names here need to match the group names
// returned by the armada-perf-client2 versions call.
const platformKubernetes = "kubernetes"
const platformOpenshift = "openshift"

const slackTokenEnvVar = "ARGONAUTS_ARM_PERF_ALERTS_SLACK_OAUTH_TOKEN"
const slackChannelID = "G5CNHCJ7R" // armada-perf-private

func main() {

	versionsJSONStr := flag.String("versionsJSON", "", "JSON file containing apc2 versions output")

	flag.Parse()

	if versionsJSONStr != nil && *versionsJSONStr == "" {
		log.Fatal("No versions JSON supplied. Use --versionsJSON parameter to supply file path.")
	}

	// Read contents of JSON file
	jsonBytes, err := ioutil.ReadFile(*versionsJSONStr)
	if err != nil {
		log.Fatal("Error reading supplied JSON file", err)
	}

	// Parse our JSON file
	var supportedVersions map[string][]map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &supportedVersions); err != nil {
		log.Fatal("Error parsing supplied JSON", err)
	}

	// Get our config for Influx
	var metricsCfg metrics.ServiceConfig
	ok := true
	if metricsCfg, ok = metrics.ReadMetricsTomlFile(); !ok {
		log.Fatal("Unable to read metrics configuration file")
	}

	// Read our stored defaults from Influx
	storedDefaults, err := readStoredDefaults(metricsCfg)
	if err != nil {
		log.Fatal("Error reading stored defaults", err)
	}

	// Initialise our platform defaults.
	platforms := map[string]versionDefaults{
		platformKubernetes: {},
		platformOpenshift:  {},
	}

	// Iterate over our platforms to set their defaults
	for platform, defaults := range platforms {

		// Get current default from supplied versions output
		for _, supportedVersion := range supportedVersions[platform] {
			// Is this version the default?
			if supportedVersion["default"] == true {
				// Save as our current default
				defaults.currentDefault = fmt.Sprintf("%v.%v", supportedVersion["major"], supportedVersion["minor"])

				break
			}
		}

		// Set our stored default
		defaults.storedDefault = storedDefaults[platform]

		// Update the values in our map. Needed due to
		// the strange way in which go maps behave.
		platforms[platform] = defaults

		// Has our default changed
		if defaults.currentDefault != defaults.storedDefault {
			// Update our stored value
			err = writeStoredDefault(metricsCfg, platform, defaults.currentDefault)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	// Do we need to output a Slack message?
	sendSlackMessage(platforms)
}

var measurementName = "platformDefaults"

func readStoredDefaults(cfg metrics.ServiceConfig) (storedDefaults map[string]string, err error) {

	// Create our client to call Influx
	influxClient, err := createInfluxClient(cfg)
	if err != nil {
		return nil, err
	}
	defer influxClient.Close()

	storedDefaults = map[string]string{
		platformKubernetes: "",
		platformOpenshift:  "",
	}

	// The last function only seems to return a single latest "row" for the
	// whole measurement so we need to query twice. Once for each platform.
	for platform := range storedDefaults {
		sql := fmt.Sprintf("SELECT last(platformDefault), platformName FROM %v WHERE platformName='%v'", measurementName, platform)
		if cfg.Metrics.Verbose {
			log.Printf("Querying InfluxDB for current default : %s", sql)
		}

		q := client.NewQuery(sql, cfg.Metrics.InfluxdbName, "")
		if response, err := influxClient.Query(q); err == nil && response.Error() == nil {
			if len(response.Results[0].Series) > 0 {
				influxResults := response.Results[0].Series[0].Values

				// Save the result in our default map
				for _, v := range influxResults {
					storedDefaults[platform] = v[1].(string)
				}
			}
		} else {
			if err != nil {
				return nil, fmt.Errorf("Error running InfluxDB query for %v: %w", platform, err)
			}
		}
	}

	return
}

func writeStoredDefault(cfg metrics.ServiceConfig, platform string, newDefault string) error {

	// Create our client to call Influx
	influxClient, err := createInfluxClient(cfg)
	if err != nil {
		return err
	}
	defer influxClient.Close()

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  cfg.Metrics.InfluxdbName,
		Precision: "s",
	})
	if err != nil {
		return fmt.Errorf("Failed to create InfluxDB batch point. Error %w", err)
	}

	tags := make(map[string]string)
	tags["platformName"] = platform

	fields := make(map[string]interface{})
	fields["platformDefault"] = newDefault

	timestamp := time.Now()
	pt, err := client.NewPoint(measurementName, tags, fields, timestamp)
	if err != nil {
		return fmt.Errorf("Error creating InfluxDB data point: %w", err)
	}
	bp.AddPoint(pt)

	if cfg.Metrics.Verbose {
		log.Printf("Writing new default to InfluxDB : %v", pt)
	}

	// write the batch of data to Influxdb
	if err := influxClient.Write(bp); err != nil {
		return fmt.Errorf("Failed to write new default for %v to InfluxDB: %w", platform, err)
	}

	return nil
}

func createInfluxClient(cfg metrics.ServiceConfig) (influxClient client.Client, err error) {

	// Create our client to call Influx
	influxClient, err = client.NewHTTPClient(client.HTTPConfig{
		Addr:     "http://" + net.JoinHostPort(cfg.Metrics.InfluxdbHost, cfg.Metrics.InfluxdbPort),
		Username: cfg.Metrics.InfluxdbUser,
		Password: os.Getenv("METRICS_DB_KEY"), // pragma: allowlist secret
		Timeout:  300 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create InfluxDB client. Error %w", err)
	}

	return influxClient, nil
}

func sendSlackMessage(platformDefaults map[string]versionDefaults) error {

	token := os.Getenv(slackTokenEnvVar)
	if len(token) == 0 {
		log.Fatalf("Slack token not provided. Check '%s' environment variable", slackTokenEnvVar)
	}

	api := slack.New(token, slack.OptionDebug(false)) // Careful if setting the debug option to true. It will output the token.

	blocks := make([]slack.Block, 0)

	// Add title section for message
	titleSection := slack.NewHeaderBlock(slack.NewTextBlockObject(slack.PlainTextType, "Platform default versions:", false, false))
	log.Println("Platform default versions:")

	// Text Block Fields containing the platform default levels
	platforms := make([]*slack.TextBlockObject, 0)

	changeOccurred := false
	for platform, defaults := range platformDefaults {

		var message, defaultVersion string

		// Has the default changed?
		if defaults.currentDefault != defaults.storedDefault {
			// At least one change has occurred
			changeOccurred = true

			// Value has changed
			message = fmt.Sprintf("Default has changed for %v:", platform)
			defaultVersion = fmt.Sprintf("%v -> %v", defaults.storedDefault, defaults.currentDefault)

			log.Println(message, defaultVersion)
		} else {
			// No change
			message = fmt.Sprintf("Default not changed for %v:", platform)
			defaultVersion = defaults.storedDefault

			log.Println(message, defaultVersion)
		}

		// Construct the slack message line for this platform
		messageField := slack.NewTextBlockObject(slack.MarkdownType, message, false, false)
		defaultField := slack.NewTextBlockObject(slack.MarkdownType, defaultVersion, false, false)
		platforms = append(platforms, messageField)
		platforms = append(platforms, defaultField)
	}

	platformsSection := slack.NewSectionBlock(nil, platforms, nil)
	blocks = append(blocks, titleSection)
	blocks = append(blocks, platformsSection)

	//  Only actually send the message if a change has occurred
	if changeOccurred {
		_, _, err := api.PostMessage(
			slackChannelID,
			slack.MsgOptionText("Platform Default Levels", false),
			slack.MsgOptionBlocks(blocks...),
			slack.MsgOptionAsUser(false), // Add this if you want that the bot would post message as a user, otherwise it will send response using the default slackbot
		)
		if err != nil {
			log.Fatalf("Error sending message to slack : %s\n", err)
		}

		log.Printf("Message successfully sent to Slack channel %s\n", slackChannelID)
	}
	return nil
}
