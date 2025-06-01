/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2020 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package e2emetrics

type metricStruct struct {
	Name        string `json:"__name__"`
	Resource    string `json:"resource"`
	SubResource string `json:"subresource"`
	Verb        string `json:"verb"`
}

// API Server Request Count Metrics
type apiServerRequestCountMetric struct {
	metricStruct
	Client      string `json:"client"`
	Code        string `json:"code"`
	ContentType string `json:"contentType"`
}
type apiServerRequestCount struct {
	Metric apiServerRequestCountMetric `json:"metric"`
	Value  []interface{}               `json:"value"`
}

// API Server Request Latency Metrics
type apiServerRequestLatenciesSummaryMetric struct {
	metricStruct
	Quantile string `json:"quantile"`
}
type apiServerRequestLatenciesSummary struct {
	Metric apiServerRequestLatenciesSummaryMetric `json:"metric"`
	Value  []interface{}                          `json:"value"`
}

type apiServerMetrics struct {
	APIServerRequestCount            []apiServerRequestCount            `json:"apiserver_request_count"`
	APIServerRequestLatenciesSummary []apiServerRequestLatenciesSummary `json:"apiserver_request_latencies_summary"`
}
type schedulerMetrics struct{}         // Not implemented. Tests do not currently run within Armada
type clusterAutoscalerMetrics struct{} // Not implemented. Tests do not currently run within Armada

// MetricsForE2ETests defines the structure of the Kubernetes E2E Testing Metrics
type MetricsForE2ETests struct {
	APIServerMetrics         apiServerMetrics         `json:"ApiServerMetrics"`
	SchedulerMetrics         schedulerMetrics         `json:"SchedulerMetrics"`
	ClusterAutoscalerMetrics clusterAutoscalerMetrics `json:"ClusterAutoscalerMetrics"`
}
