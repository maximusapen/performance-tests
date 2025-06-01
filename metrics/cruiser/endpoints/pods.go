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

type container struct {
	Name  string `json:"name"`
	Usage usage  `json:"usage"`
}

type podMetadata struct {
	metadata
	Namespace string `json:"namespace"`
}
type pod struct {
	MetaData   podMetadata `json:"metadata"`
	Timestamp  string      `json:"timestamp"`
	Containers []container `json:"containers"`
}

// PodMetrics define data/methods for pod metrics data
type PodMetrics struct {
	Pods []pod `json:"items"`
}

// Name returns the name of the resource (i.e. "nodes")
func (pm *PodMetrics) Name() string {
	return "pods"
}

// Metrics returns resource data via Kuberrnetes metrics api
func (pm *PodMetrics) Metrics() Resource {
	return Metrics(pm)
}

// BMMetrics returns a set of namespace/pod/container IBM Cloud monitoring service metrics
func (pm *PodMetrics) BMMetrics() []metricsservice.BluemixMetric {
	var bm []metricsservice.BluemixMetric

	type resourceUsage map[string]int64

	type item struct {
		tstamp string
		res    resourceUsage
		level  string
	}

	rm := make(map[string]item)

	levels := []string{}

	// Levels are hierarchical. Requested level and its ancestors will be collected.
	switch Level {
	case "container":
		levels = append(levels, "container")
		fallthrough
	case "pod":
		levels = append(levels, "pod")
		fallthrough
	case "namespace":
		levels = append(levels, "namespace")

	default:
		fatallog.Fatalf("Unrecognized level '%s'\n", Level)
	}

	for _, p := range pm.Pods {
		for _, c := range p.Containers {
			var rid string
			var ts string

			// Some containers may have activities so low that cpu usage is not recorded.
			// This is handled in resources.go.  Commented out related logs below
			// in case we want to turn it back on

			//if c.Usage.CPU == "" {
			// Some containers may have activities so low that cpu usage is not recorded.
			// Setting to 0 if no c.Usage.CPU
			//log.Printf("No c.Usage.CPU data for pod %v container %v \n", p, c)
			//}

			cpu := parseMetric("cpu", c.Usage.CPU)
			mem := parseMetric("mem", c.Usage.Memory)

			for _, l := range levels {
				switch l {
				case "namespace":
					ts = p.MetaData.CreationTime
					rid = p.MetaData.Namespace

				case "pod":
					ts = p.MetaData.CreationTime
					rid = strings.Join([]string{p.MetaData.Namespace, p.MetaData.Name}, ".")

				case "container":
					ts = p.Timestamp
					rid = strings.Join([]string{p.MetaData.Namespace, p.MetaData.Name, c.Name}, ".")
				}

				if re == nil || re.MatchString(rid) {
					if rm[rid].res == nil {
						ru := make(resourceUsage)
						rm[rid] = item{tstamp: ts, res: ru, level: l}
					}
					rm[rid].res["cpu"] += cpu
					rm[rid].res["mem"] += mem
				}
			}
		}
	}

	// Level
	for rn, rv := range rm {
		// Metrics (e.g. cpu and memory)
		for ln, lv := range rv.res {
			var div float64
			switch ln {
			case "cpu":
				// Report CPU in milli-cores
				div = 1e6
			case "mem":
				// Report memory in bytes
				div = 1.0
			}

			ts, err := time.Parse(time.RFC3339, rv.tstamp)
			if err != nil {
				fatallog.Fatalf("Unable to parse metric timestamp for resource %s\n : %s", ln, err.Error())
			}

			mn := fmt.Sprintf("cruiser-%s-metrics", rv.level)
			mn = strings.Join([]string{mn, rn, ln, "sparse-avg"}, ".")

			// Include the test name (if specified) at the start of the metric name
			if len(Testname) > 0 {
				mn = strings.Join([]string{Testname, mn}, ".")
			}

			bm = append(bm, metricsservice.BluemixMetric{Name: mn, Timestamp: ts.Unix(), Value: float64(lv) / div})
		}
	}
	return bm
}

// Unmarshal returns the name of the resource (i.e. "nodes")
func (pm *PodMetrics) Unmarshal(data []byte) error {
	return json.Unmarshal(data, &pm)
}

func (pm *PodMetrics) String() string {
	str := ""
	for _, p := range pm.Pods {
		str += fmt.Sprintf("%s\n\t%s: %s\n", p.MetaData.CreationTime, p.MetaData.Namespace, p.MetaData.Name)
		for _, c := range p.Containers {
			str += fmt.Sprintf("\t\t%s - CPU: %s, Mem: %s\n", c.Name, c.Usage.CPU, c.Usage.Memory)
		}
	}
	return str
}
