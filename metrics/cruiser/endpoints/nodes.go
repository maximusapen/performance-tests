/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2018, 2020 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package endpoints

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	metricsservice "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"
)

type node struct {
	MetaData  metadata `json:"metadata"`
	Timestamp string   `json:"timestamp"`
	Usage     usage    `json:"usage"`
}

// NodeMetrics define data/methods for node metrics data
type NodeMetrics struct {
	Nodes []node `json:"items"`
}

// Name returns the name of the resource type (i.e. "nodes")
func (nm *NodeMetrics) Name() string {
	return "nodes"
}

// Metrics gathers resource metrics data via the Kuberrnetes metrics api
func (nm *NodeMetrics) Metrics() Resource {
	return Metrics(nm)
}

// BMMetrics returns a set of resource metrics for sending to IBM Cloud Monitoring service
func (nm *NodeMetrics) BMMetrics() []metricsservice.BluemixMetric {
	var bm []metricsservice.BluemixMetric

	// For each node in the cluster
	for _, n := range nm.Nodes {
		// Create a suitable name prefix, replacing any periods (e.g. from ip address) which aren't supported in names
		mn := strings.Join([]string{"cruiser-node-metrics", strings.Replace(n.MetaData.Name, ".", "_", -1)}, ".")

		// Include the test name (if specified) at the start of the metric name
		if len(Testname) > 0 {
			mn = strings.Join([]string{Testname, mn}, ".")
		}

		// We need the timestamp in Unix time, so parse it
		ts, err := time.Parse(time.RFC3339, n.Timestamp)
		if err != nil {
			fatallog.Fatalf("Unable to parse metric timestamp for node %s\n : %s", n.MetaData.Name, err.Error())
		}

		// We stored cpu metrics in nano-cores (because that's the resolution used by Kubernetes), but we'll report in milli-cores
		cpu := float64(parseMetric("cpu", n.Usage.CPU)) / 1e6

		// Memory reported in bytes.
		mem := float64(parseMetric("mem", n.Usage.Memory))

		bm = append(bm,
			metricsservice.BluemixMetric{
				Name:      strings.Join([]string{mn, "cpu", "sparse-avg"}, "."),
				Timestamp: ts.Unix(),
				Value:     cpu,
			})

		bm = append(bm,
			metricsservice.BluemixMetric{
				Name:      strings.Join([]string{mn, "mem", "sparse-avg"}, "."),
				Timestamp: ts.Unix(),
				Value:     mem,
			})
	}
	return bm
}

// Unmarshal returns the name of the resource (i.e. "nodes")
func (nm *NodeMetrics) Unmarshal(data []byte) error {
	return json.Unmarshal(data, &nm)
}

func (nm *NodeMetrics) String() string {
	str := ""
	for _, n := range nm.Nodes {
		str += fmt.Sprintf("%s\n\t%s - CPU: %s, Mem: %s\n", n.Timestamp, n.MetaData.Name, n.Usage.CPU, n.Usage.Memory)
	}
	return str
}
