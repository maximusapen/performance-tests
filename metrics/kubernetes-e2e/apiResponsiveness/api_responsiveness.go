/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package apiresponsiveness

type latencyDataItems struct {
	Data   map[string]float64 `json:"data"`
	Unit   string             `json:"unit"`
	Labels map[string]string  `json:"labels"`
}

// Latency defines the strcuture of the Kubernetes E2E Testing - "Density Pod Startup Latency" Metrics
type Latency struct {
	Version   string             `json:"version"`
	DataItems []latencyDataItems `json:"dataItems"`
}
