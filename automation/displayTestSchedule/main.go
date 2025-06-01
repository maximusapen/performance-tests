/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2021, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

/*
 * The Generate-Results-Visualization jenkins job calls parseSchedule.sh reconfigure the automation schedule
 * into a form that is easly parsable into tables. The results are output to /tmp/perfAutomationTests.json.
 * This program builds the tests and clients tables that highlight the test table from two perspectives.
 */

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

var jenkinsActivateDeactivateTestClients = "https://alchemy-testing-jenkins.swg-devops.com/view/Armada-performance/job/Armada-Performance/job/Automation/job/Activate-Deactivate-Test-Clients/build"
var jenkinsEnableDisableTests = "https://alchemy-testing-jenkins.swg-devops.com/view/Armada-performance/job/Armada-Performance/job/Automation/job/Enable-Disable-Tests/build"

// Day Data on the build for that day
type Day struct {
	Day          string        `json:"day"`
	Client       string        `json:"client"`
	Enable       bool          `json:"enable"`
	Tests        []ClientTests `json:"tests"`
	ClientActive bool
}

// ClientTests ...
type ClientTests struct {
	Name     string `json:"test"`
	Enable   bool   `json:"enable"`
	TestHour int    `json:"test_hour"`
}

// Client ...
type Client struct {
	Name      string `json:"client"`
	ShortName string `json:"shortName"`
	Active    bool   `json:"active"`
	Days      []Day  `json:"days"`
}

// Test ...
type Test struct {
	Name  string `json:"test"`
	Owner string `json:"owner"`
	Days  []Day  `json:"days"`
}

// Tests ...
type Tests struct {
	Tests   []Test   `json:"tests"`
	Clients []Client `json:"clients"`
}

var tests Tests

var weekdays = []time.Weekday{
	time.Sunday,
	time.Monday,
	time.Tuesday,
	time.Wednesday,
	time.Thursday,
	time.Friday,
	time.Saturday,
}
var automationSetup = "cluster setup"

const dayInMilliseconds = 24 * 60 * 60 * 1000
const weekInMilliseconds = 7 * dayInMilliseconds

var testScheduleTemplate = `<style>table { border-collapse: collapse; }
table, th, td { border: 1px solid black; }
tr:nth-child(even) {background-color: #f2f2f2;}
tr:hover {background-color:#d6d6d6;}
</style>
<table>
<tr><td><b>Test</b></td><td><b>Owner</b></td><td><b>Monday</b></td><td><b>Tuesday</b></td><td><b>Wednesday</b></td><td><b>Thursday</b></td><td><b>Friday</b></td><td><b>Saturday</b></td><td><b>Sunday</b></td></tr>
{{range .Tests}}
<tr>
<td><b>{{.Name}}</b></td>
<td>{{.Owner}}</td>
{{range .Days}}
  {{if .Client}}
    {{if .ClientActive}}
      {{if not .Enable}}
        <td style="color:grey;">
      {{else}}
        <td>
      {{end}}
    {{else}}
      {{if not .Enable}}
        <td style="color:grey; background-color:lightyellow;">
      {{else}}
        <td style="background-color:lightyellow;">
      {{end}}
    {{end}}
    {{.Client}}
  {{else}}
    <td>
  {{end}}
  </td>
  </td>
{{end}}
</tr>
{{end}}</table>`

var clientScheduleTemplate = `<table>
<tr><td><b>Client</b></td><td><b>Monday</b></td><td><b>Tuesday</b></td><td><b>Wednesday</b></td><td><b>Thursday</b></td><td><b>Friday</b></td><td><b>Saturday</b></td><td><b>Sunday</b></td></tr>
{{range .Clients}}
{{if .Active}}<tr>{{else}}<tr style="background-color:lightyellow;">{{end}}
<td><b>{{.Name}}</b></td>
{{range .Days}}<td>{{if .Tests}}{{range .Tests}}{{if not .Enable}}<p style="color:grey">{{else}}<p style="color:black">{{end}}{{.Name}} ({{.TestHour}}:00)</p>{{end}}{{end}}{{end}}</td>
</tr>
{{end}}
</table>`

var keyTemplate = `<b>Key</b><table>
<tr><td><td><b>Test Enabled</b></td><td><b>Test Disabled</b></td></tr>
<tr><td><b>Client Enabled</b></td><td>Test/Client</td><td style="color:grey;">Test/Client</td></tr>
<tr style="background-color:lightyellow;"><td><b>Client Disabled</b></td><td>Test/Client</td><td style="color:grey;">Test/Client</td></tr>
</table><br>`

func getClientActive(clientName string) bool {
	for _, client := range tests.Clients {
		if strings.Contains(client.ShortName, clientName) {
			return client.Active
		}
	}
	return false
}

var defaultKubeVersion string
var defaultOpenshiftVersion string

func main() {
	testFile, err := os.Open("/tmp/perfAutomationSchedule.json")

	if err != nil {
		fmt.Println(err)
	}
	defer testFile.Close()

	// Parse flags for default versions
	flag.StringVar(&defaultKubeVersion, "defaultKubeVersion", "", "The current default Kube version")
	flag.StringVar(&defaultOpenshiftVersion, "defaultOpenshiftVersion", "", "The current default Openshift version")
	flag.Parse()

	fmt.Println("<html><body>")

	fmt.Println("<a href=\"./automation.html\">Test Results</a><br><br>")
	timeStamp := time.Now().Format("Mon Jan 02 15:04:05 MST 2006")
	fmt.Println("<b>Report generated</b>:", timeStamp, "<br>")
	fmt.Println(keyTemplate)

	fmt.Println("<b>Current Kube default version</b>:", defaultKubeVersion, "<br>")
	fmt.Println("<b>Current Openshift default version</b>:", defaultOpenshiftVersion, "<br><br>")

	testBytes, _ := ioutil.ReadAll(testFile)

	err = json.Unmarshal(testBytes, &tests)
	if err != nil {
		panic(err)
	}

	for i, test := range tests.Tests {
		for j, day := range test.Days {
			tests.Tests[i].Days[j].ClientActive = getClientActive(day.Client)
		}
	}

	fmt.Println("<a href=\"" + jenkinsEnableDisableTests + "\" target=\"_blank\">Enable-Disable Tests</a><br>")

	t := template.New("t")
	testParser, err := t.Parse(testScheduleTemplate)
	if err != nil {
		panic(err)
	}

	err = testParser.Execute(os.Stdout, tests)
	if err != nil {
		panic(err)
	}

	fmt.Println("<br><a href=\"" + jenkinsActivateDeactivateTestClients + "\" target=\"_blank\">Activate-Deactivate-Test-Clients</a>")
	c := template.New("c")
	clientParser, err := c.Parse(clientScheduleTemplate)
	if err != nil {
		panic(err)
	}

	err = clientParser.Execute(os.Stdout, tests)
	if err != nil {
		panic(err)
	}

	fmt.Println("</body></html>")
}
