/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2019, 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package metrics

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli"
	"github.ibm.com/alchemy-containers/armada-performance/armada-perf-client2/models"
	metricsservice "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"
)

var debug = true

type stateTimes struct {
	minTime   time.Duration
	maxTime   time.Duration
	totalTime time.Duration
}

// WorkerStateTimes tracks worker state transition times
// Map: Key is worker state (e.g. provisioning, reloading, deploying, etc.)
type WorkerStateTimes map[string]time.Duration

// Data holds metrics information associated with a command and a set of cluster workers
type Data struct {
	Enabled bool
	Command string
	W       Workers
	C       Clusters
	L       Locations
	H       Hosts
	P       WorkerPools
	E       Endpoints
	Data    map[string]interface{}
}

// Clusters holds metrics data for a set of clusters - assume multiple clusters have the same number of workers
type Clusters struct {
	WorkerCount int
	Duration    []time.Duration
}

// Locations holds metrics data for a set of locations
type Locations map[string]*Location

// Location holds metrics data for a single location
type Location struct {
	CoreOS   bool
	Duration time.Duration
}

// Endpoints holds metrics data for a set of locations
type Endpoints []time.Duration

// Workers holds metrics data for a set of workers
// Map: Key is worker id
type Workers map[string]*Worker

// Worker holds metrics data for a worker
type Worker struct {
	CurState  string
	CurStatus string
	Durations WorkerStateTimes
	Failed    bool
}

// WorkerPools holds metrics data for a set of worker pools
type WorkerPools []time.Duration

// Hosts holds metrics data fot a set of Satellite hosts
type Hosts struct {
	Location Location
	Hosts    map[string]*Host
}

// Host holds metrics data fot a Satellite host
type Host struct {
	State    string
	Message  string
	Duration time.Duration
}

// Initialize performs any required initialization before metrics are gathered
func Initialize(c *cli.Context) error {
	md := c.App.Metadata[models.MetricsFlagName].(*Data)
	md.Data = make(map[string]interface{})
	return nil
}

// WriteMetrics sends results to our metrics service
func WriteMetrics(c *cli.Context) error {
	md := c.App.Metadata[models.MetricsFlagName].(*Data)
	if md.Enabled {
		var totalDeployFailures int

		if debug {
			for wid, states := range md.W {
				fmt.Fprintln(c.App.Writer, wid)
				for s, d := range states.Durations {
					fmt.Fprintf(c.App.Writer, "\t%s\t%v\n", s, d)
				}
			}
		}

		sm := make(map[string]*stateTimes)
		for _, w := range md.W {
			for s, d := range w.Durations {
				if val, ok := sm[s]; ok {
					if d < val.minTime {
						val.minTime = d
					}
					if d > val.maxTime {
						val.maxTime = d
					}
					val.totalTime += d
				} else {
					sm[s] = new(stateTimes)
					sm[s].minTime = d
					sm[s].maxTime = d
					sm[s].totalTime = d
				}
			}

			if w.Failed {
				totalDeployFailures++
			}
		}

		var bm []metricsservice.BluemixMetric

		// Some metric names should be differentiated on number of workers.
		// First check the Worker metrics, and secondly check cluster metrics
		// (N.B. we're assuming multiple clusters have the same number of workers)
		workerCount := len(md.W)
		if workerCount == 0 {
			workerCount = md.C.WorkerCount
		}

		metricsPrefix := "armada-perf-client2"
		if workerCount > 0 {
			resourceCountStr := "workers_" + strconv.Itoa(workerCount-totalDeployFailures)
			metricsPrefix = strings.Join([]string{metricsPrefix, resourceCountStr}, ".")
		}
		metricsPrefix = strings.Join([]string{metricsPrefix, md.Command}, ".")

		// Generate min, mean and max state transition times across all workers.
		for state, d := range sm {
			bm = append(bm, metricsservice.BluemixMetric{
				Name:  strings.Join([]string{metricsPrefix, state, "duration", "min"}, "."),
				Value: d.minTime.Seconds(),
			})
			bm = append(bm, metricsservice.BluemixMetric{
				Name:  strings.Join([]string{metricsPrefix, state, "duration", "max"}, "."),
				Value: d.maxTime.Seconds(),
			})
			bm = append(bm, metricsservice.BluemixMetric{
				Name:  strings.Join([]string{metricsPrefix, state, "duration", "mean"}, "."),
				Value: d.totalTime.Seconds() / float64(workerCount),
			})
		}

		// Add worker deploy failures if appropriate
		if workerCount > 0 {
			bm = append(bm, metricsservice.BluemixMetric{
				Name:  strings.Join([]string{metricsPrefix, "worker-failures"}, "."),
				Value: totalDeployFailures,
			})
		}

		// Process cluster metrics
		if len(md.C.Duration) > 0 {
			bm = processDurations(bm, md.C.Duration, metricsPrefix)
		}

		// Process Worker Pool metrics
		if len(md.P) > 0 {
			bm = processDurations(bm, md.P, metricsPrefix)
		}

		// Process Satellite Location metrics
		if len(md.L) > 0 {
			ld := make([]time.Duration, 0, len(md.L))
			mp := metricsPrefix
			for _, l := range md.L {
				if l.CoreOS {
					mp = strings.Join([]string{metricsPrefix, "coreos"}, ".")
				}
				ld = append(ld, l.Duration)
			}

			bm = processDurations(bm, ld, mp)
		}

		// Process Satellite Endpoint metrics
		if len(md.E) > 0 {
			bm = processDurations(bm, md.E, metricsPrefix)
		}

		// Process Satellite Host metrics
		if len(md.H.Hosts) > 0 {
			mp := metricsPrefix
			if md.H.Location.CoreOS {
				mp = strings.Join([]string{metricsPrefix, "coreos"}, ".")
			}
			hd := make([]time.Duration, 0, len(md.H.Hosts))
			for _, h := range md.H.Hosts {
				hd = append(hd, h.Duration)
			}
			bm = processDurations(bm, hd, mp)
		}
		if md.H.Location.Duration != 0 {
			mp := metricsPrefix
			if md.H.Location.CoreOS {
				mp = strings.Join([]string{metricsPrefix, "coreos"}, ".")
			}
			bm = processDurations(bm, []time.Duration{md.H.Location.Duration}, strings.Join([]string{mp, "location"}, "."))
		}

		// Finally add in any command specific metrics
		for mn, mv := range md.Data {
			bm = append(bm, metricsservice.BluemixMetric{
				Name:  strings.Join([]string{metricsPrefix, mn}, "."),
				Value: mv,
			})
		}

		if debug {
			fmt.Fprintln(c.App.Writer, bm)
		}

		// Send to Influx DB
		metricsservice.WriteBluemixMetrics(bm, true, "apc2", "")
	}
	return nil
}

func processDurations(bm []metricsservice.BluemixMetric, md []time.Duration, mp string) []metricsservice.BluemixMetric {
	var cd stateTimes

	if len(md) == 0 {
		return bm
	}

	for i, d := range md {
		if i == 0 || d > cd.maxTime {
			cd.maxTime = d
		}
		if i == 0 || d < cd.minTime {
			cd.minTime = d
		}
		cd.totalTime += d
	}

	bm = append(bm, metricsservice.BluemixMetric{
		Name:  strings.Join([]string{mp, "duration", "min"}, "."),
		Value: cd.minTime.Seconds(),
	})
	bm = append(bm, metricsservice.BluemixMetric{
		Name:  strings.Join([]string{mp, "duration", "max"}, "."),
		Value: cd.maxTime.Seconds(),
	})
	bm = append(bm, metricsservice.BluemixMetric{
		Name:  strings.Join([]string{mp, "duration", "mean"}, "."),
		Value: cd.totalTime.Seconds() / float64(len(md)),
	})

	return bm
}
