/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2018, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package metricsservice

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

type alertsTomlConfig struct {
	Config  alertsMetadata
	Alerts  map[string]alertsConfig
	Carrier carrierIDConfig
}

type alertsMetadata struct {
	ChartsURL         string `toml:"charts_url"`
	ChartsAccess      string `toml:"charts_access"`
	BaseRazeeURL      string `toml:"base_razee_url"`
	RazeeUserID       string `toml:"armada_perf_razeeUserId"`
	AlertsActive      bool   `toml:"alerts_active"`
	SendAlertsToRazee bool   `toml:"send_alerts_to_razee"`
	Verbose           bool   `toml:"verbose"`
}

type alertsConfig struct {
	TestDetail  string      `toml:"test_detail"`
	AlertDetail [][4]string `toml:"alert_detail"`
}

type carrierIDConfig struct {
	CarrierID string `toml:"carrier_id"`
}

// RazeeDash structure
type razeeDash struct {
	Logs string `json:"logs"`
	Pass bool   `json:"pass"`
}

// WriteRazeeDashData checks metrics values against alerts.toml. Send pass/fail alerts to RazeeDash as appropriate.
func WriteRazeeDashData(metrics []BluemixMetric, razeedashAPIKey string, testName string) {
	var (
		alertsTomlCfg alertsTomlConfig
		overallPass   bool
		razeeRequest  razeeDash
		carrierID     string
		verbose       bool
		metricsToSend bool
		sendToRazee   bool
		fullRazeeURL  string
	)

	//Only send to RazeeDash if the armada_performance_razeedash_api_key is set
	if len(razeedashAPIKey) == 0 {
		razeedashAPIKey = os.Getenv("RAZEE_API_KEY")
		if verbose {
			log.Println("Getting RAZEE_API key from env var")
			if len(razeedashAPIKey) == 0 {
				log.Println("No RazeeDash api-key key found. If calling from Jenkins, this may mean the SEND_METRICS_TO_RAZEEDASH boolean is set to false. No alert data will be sent to RazeeDash.")
			} else if len(razeedashAPIKey) > 4 {
				log.Printf("Razee API Key (first 4 chars) = %s.\n", razeedashAPIKey[0:3])
			}
		}
	}
	if len(razeedashAPIKey) == 0 {
		return
	}

	if len(testName) == 0 {
		testName = os.Getenv("TEST_NAME")
		if verbose {
			log.Println("Getting TEST_NAME from env var")
		}
		if len(testName) == 0 {
			log.Println("No test name specified. No alert data will be sent to RazeeDash.")
			return
		}
	}
	log.Printf("Test name: %s\n", testName)

	if _, err := toml.DecodeFile(filepath.Join(getPerfRepoPath(), "metrics", "bluemix", "alerts.toml"), &alertsTomlCfg); err != nil {
		log.Printf("Error parsing 'alerts toml' config file: %s\n", err.Error())
		return
	}
	if !alertsTomlCfg.Config.AlertsActive {
		log.Printf("Alerts turned off in 'alerts.toml'. No alerts will be sent to RazeeDash")
		return
	}

	verbose = alertsTomlCfg.Config.Verbose
	sendToRazee = alertsTomlCfg.Config.SendAlertsToRazee
	carrierID = alertsTomlCfg.Carrier.CarrierID
	if len(carrierID) == 0 {
		log.Println("No CarrierId found in alerts.toml. No alert data will be sent to RazeeDash.")
		return
	}

	log.Printf("CarrierID found: %s\n", carrierID)

	// Append the kube version to the test name if it is known
	ksv := strings.Split(os.Getenv("K8S_SERVER_VERSION"), "_")[0]
	if len(ksv) > 0 {
		fullRazeeURL = alertsTomlCfg.Config.BaseRazeeURL + "/" + carrierID + "/" + testName + "_____(K8s: " + ksv + ")/test_results"
	} else {
		fullRazeeURL = alertsTomlCfg.Config.BaseRazeeURL + "/" + carrierID + "/" + testName + "/test_results"
	}

	testSpecificAlerts := alertsTomlCfg.Alerts[testName].AlertDetail

	if verbose {
		log.Printf("alerts.WriteRazeeDashData: carrierID =%v razeeURL=%s", carrierID, fullRazeeURL)
		log.Printf("\nAlerts found in alerts.toml for test %s:", testName)
		for _, anAlert := range testSpecificAlerts {
			log.Printf("Alert = %s %s %s %s", anAlert[0], anAlert[1], anAlert[2], anAlert[3])
		}
	}

	overallPass = true
	metricsToSend = false
	var logsBuffer bytes.Buffer
	var oneTestpass = true
	logsBuffer.WriteString("\n--- Test Name   : " + testName + "\n\n")
	logsBuffer.WriteString("--- Test Date   : " + (time.Now().UTC().Format("Mon _2 Jan 2006 : 15:04:05") + " UTC \n\n"))
	logsBuffer.WriteString("--- Test Detail : " + alertsTomlCfg.Alerts[testName].TestDetail + "\n\n\n")

	for _, ametric := range metrics {

		for _, anAlert := range testSpecificAlerts {
			if strings.Contains(ametric.Name, anAlert[0]) {

				metricsToSend = true
				oneTestpass = false

				log.Println("\nAlert match found:")
				log.Printf("Metric = %s;\n     Value = %s\n", ametric.Name, ametric.Value)
				log.Printf("Alert = %s;\n     Ceiling = %s;\n     Value = %s\n", anAlert[0], anAlert[1], anAlert[2])

				upperLimit, err := strconv.ParseBool(anAlert[1])
				if err != nil {
					log.Printf("alerts.WriteRazeeDashData: Failed to parse alert ceiling boolean\n" + anAlert[1] + "  " + err.Error())
					return
				}

				alertLimit, err := strconv.ParseFloat(anAlert[2], 64)
				if err != nil {
					log.Printf("alerts.WriteRazeeDashData: Failed to parse alert float value: \n" + anAlert[2] + "  " + err.Error())
					return
				}

				metricFloatValue, err := interfaceToFloat64(ametric.Value)
				if err != nil {
					log.Printf("alerts.WriteRazeeDashData: \n" + err.Error())
					return
				}

				if upperLimit && (metricFloatValue < alertLimit) {
					oneTestpass = true
				} else if !upperLimit && (metricFloatValue > alertLimit) {
					oneTestpass = true
				}

				if !oneTestpass {
					overallPass = false
				}

				logsBuffer.WriteString("----- Alerts Examined -----\n")
				logsBuffer.WriteString("Alert Description : " + anAlert[3] + "\n")
				logsBuffer.WriteString("Alert Metric : " + anAlert[0] + "\n")
				if upperLimit {
					logsBuffer.WriteString("Alert CEILING Threshold:" + anAlert[2] + "\n")
				} else {
					logsBuffer.WriteString("Alert FLOOR Threshold:" + anAlert[2] + "\n")
				}
				logsBuffer.WriteString("Measured Value = " + strconv.FormatFloat(metricFloatValue, 'f', -1, 64) + "\n")
				logsBuffer.WriteString("Pass = " + strconv.FormatBool(oneTestpass) + "\n\n")
			}
		}
	}

	if sendToRazee && metricsToSend {

		logsBuffer.WriteString("Overall test result for " + testName + " = ")
		if overallPass {
			logsBuffer.WriteString("PASS\n")
		} else {
			logsBuffer.WriteString("FAIL\n")
		}

		logsBuffer.WriteString("\n\n\n\n---------------------------------------------\n\n")
		logsBuffer.WriteString(alertsTomlCfg.Config.ChartsURL + "\n")
		logsBuffer.WriteString(alertsTomlCfg.Config.ChartsAccess + "\n\n")

		finalResultBody := logsBuffer.String()
		log.Println(finalResultBody)

		razeeRequest.Pass = overallPass // pragma: allowlist secret
		razeeRequest.Logs = finalResultBody
		razeeRequestBody, err := json.MarshalIndent(razeeRequest, "", "  ")
		if err != nil {
			log.Printf("Failed to set RazeeDash alert body. No alerts will be sent to RazeeDash.\n" + err.Error())
			return
		}

		// Only going to send failures to Razee to reduce noise
		if !overallPass {
			sendToRazeeDash(overallPass, alertsTomlCfg.Config.RazeeUserID, razeedashAPIKey, fullRazeeURL, razeeRequestBody, verbose)
		}

	} else if !sendToRazee {
		log.Printf("\nalerts.toml:SendAlertsToRazee is set to false so no alerts will be sent to RazeeDash.\n")
	}

}

func sendToRazeeDash(pass bool, razeeUserID string, razeeAPIKey string, fullRazeeURL string, razeeRequestBody []byte, verbose bool) {
	//	Creates the following request:
	//  curl -H "Content-Type: application/json" -H "x-user-id: xxx" -H "x-api-key: xxx" -d '{"logs": "A test","pass": "false"}' -X POST https://razeedash.oneibmcloud.com/api/v1/clusters/<cluster-id>/armada-performance-http-test/test_results   // pragma: allowlist secret

	req, err := http.NewRequest("POST", fullRazeeURL, bytes.NewReader(razeeRequestBody))
	if err != nil {
		log.Println("Failed to generate Metrics (Bad Request). No metrics data will be available\n" + err.Error())
		return
	}

	req.Header.Set("x-api-key", razeeAPIKey)
	req.Header.Set("x-user-id", razeeUserID)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Failed to send to RazeeDash : (Request Failed). \n" + err.Error())
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		// Assume not fatal, so any remaining tests will be attempted
		log.Printf("Failed to send to RazeeDash. Response Status Code: %s.\n", resp.Status)
		respData, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read response body: %s\n", err.Error())
			return
		}

		log.Println(string(respData))
	}
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("\nFailed to read Razee response data %s\n", err.Error())
		return
	}

	log.Println("\n" + string(respData))
}
