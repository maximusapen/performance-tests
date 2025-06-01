/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2020 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package apiresponsivenessoverall

type overallLatencyResults struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

type overallLatencyData struct {
	ResultType string                  `json:"resultType"`
	Result     []overallLatencyResults `json:"result"`
}

// OverallLatency defines the structure of the overall latency metrics obtained from Prometheus at the end of the test using curl
type OverallLatency struct {
	Status string             `json:"status"`
	Data   overallLatencyData `json:"data"`
}
