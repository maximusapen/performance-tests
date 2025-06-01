
package metrics

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/config"
	metricsservice "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"
)

// WorkerMetrics tracks worker creation times
type WorkerMetrics struct {
	MetricTime     int64
	Duration       time.Duration
	WorkersCreated int
}

// WorkerStateMetrics tracks worker state transition times
// Map: Key is worker state (e.g. provisioning, reloading, deploying, etc.)
type WorkerStateMetrics map[string]time.Duration

// WorkerStates tracks state transition times for a single worker
type WorkerStates struct {
	CurState  string
	CurStatus string
	Metrics   WorkerStateMetrics
}

// ClusterWorkerStateMetrics tracks worker state durations for a cluster
// Map: Key is worker id
type ClusterWorkerStateMetrics map[string]WorkerStates

// RequestMetric defines the metrics data we care about for an API request/action
type RequestMetric struct {
	ClusterName         string
	ResponseTime        time.Duration
	ActionTime          time.Duration
	Workers             []WorkerMetrics
	WorkerCreationTimes map[string]float64
	ClusterWorkerStates ClusterWorkerStateMetrics
	ActionFailed        bool
	BackendFailed       bool
}

// ArmadaMetrics defines the metrics details
type ArmadaMetrics []RequestMetric

func percentile(values []float64, pcnt int) (float64, error) {
	if pcnt < 0 || pcnt > 100 {
		return 0.0, fmt.Errorf("Invalid percentile %d requested", pcnt)
	}

	var pcntVal float64
	numValues := len(values)
	if (numValues > 1) && (math.Remainder(float64(pcnt)*float64(numValues), 100.0) == 0) {
		pos := pcnt * numValues / 100
		lower := values[pos-1]
		upper := values[pos]
		pcntVal = (lower + upper) / 2.0
	} else {
		pcntVal = values[int(math.Ceil((float64(pcnt)/100.0)*float64(numValues)))-1]
	}
	return pcntVal, nil
}

// WriteArmadaMetrics sends the results to the Bluemix metrics service
func WriteArmadaMetrics(action config.ActionType, workerCount int, metrics *ArmadaMetrics, testName string, dbKey string) {

	apiRequestCount := len(*metrics)

	if !(apiRequestCount > 0) {
		fmt.Println("Metrics data not supplied. No metrics data will be available")
		return
	}
	actionStr := action.String()

	var resourceCountStr = ""
	if action.WorkerCreation() {
		if workerCount == -1 {
			workerCount = 0
		}
		resourceCountStr += "workers_" + strconv.Itoa(workerCount)
	}

	metricsPrefix := strings.Join([]string{"armada_api", resourceCountStr, actionStr}, ".")

	// We're supplied a slice of response and action times for each Armada API request.
	// The request time is how long it takes armada-api to respond to our request
	// The action time is how long it takes to complete the action.
	// For the action time to make sense we need to be running in blocking mode and polling for worker status information.
	// Each request will be of the same type, e.g. CreateCluster, GetClusterWorkers, etc.
	// So, let's start by generating these one or two metrics:
	//  the name is the specified prefix from the configuration file combined with the number and type of API request(s).
	//  the value is the mean response time across all these requests/actions
	var totalActionTime time.Duration
	var totalResponseTime time.Duration

	var bm []metricsservice.BluemixMetric

	responseTimes := make([]float64, apiRequestCount)
	actionTimes := make([]float64, apiRequestCount)
	var actionFailures int
	var backendFailures int
	var firstAction int
	var clusterNames []string

	for i, mval := range *metrics {
		//Gather a list of cluster names - these will be removed from the influx db metric name so it is displayed correctly
		if len(mval.ClusterName) > 0 {
			if !contains(clusterNames, mval.ClusterName) {
				clusterNames = append(clusterNames, mval.ClusterName)
			}
		}
		responseTimes[i] = mval.ResponseTime.Seconds()
		if mval.ActionFailed {
			actionFailures++
			firstAction++
			if mval.BackendFailed {
				backendFailures++
			}
		} else if mval.ActionTime.Seconds() > 0 {
			actionTimes[i] = mval.ActionTime.Seconds()
			totalActionTime += mval.ActionTime
			totalResponseTime += mval.ResponseTime
		} else {
			firstAction++
		}

		for _, wm := range mval.Workers {
			bm = append(bm, metricsservice.BluemixMetric{
				Name:      strings.Join([]string{metricsPrefix, mval.ClusterName, "Worker_Count", "max"}, "."),
				Timestamp: wm.MetricTime,
				Value:     wm.WorkersCreated,
			})
		}

		type stateTimes struct {
			minTime   time.Duration
			maxTime   time.Duration
			totalTime time.Duration
		}

		sm := make(map[string]*stateTimes)
		for _, w := range mval.ClusterWorkerStates {
			for s, d := range w.Metrics {
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
		}

		totalWorkers := len(mval.WorkerCreationTimes)
		if totalWorkers > 0 {
			var totalWorkerCreationTime float64

			var minWorkerCreationTime = math.MaxFloat64
			var maxWorkerCreationTime float64
			for _, wm := range mval.WorkerCreationTimes {
				if wm < minWorkerCreationTime {
					minWorkerCreationTime = wm
				}
				if wm > maxWorkerCreationTime {
					maxWorkerCreationTime = wm
				}
				totalWorkerCreationTime += wm
			}

			meanWorkerCreationTime := totalWorkerCreationTime / float64(totalWorkers)
			bm = append(bm, metricsservice.BluemixMetric{
				Name:  strings.Join([]string{metricsPrefix, mval.ClusterName, "Min_Worker_Creation_Time", "min"}, "."),
				Value: minWorkerCreationTime,
			})
			bm = append(bm, metricsservice.BluemixMetric{
				Name:  strings.Join([]string{metricsPrefix, mval.ClusterName, "Max_Worker_Creation_Time", "max"}, "."),
				Value: maxWorkerCreationTime,
			})
			bm = append(bm, metricsservice.BluemixMetric{
				Name:  strings.Join([]string{metricsPrefix, mval.ClusterName, "Mean_Worker_Creation_Time", "sparse-avg"}, "."),
				Value: meanWorkerCreationTime,
			})

			// Generate min, mean and max state transition times across all workers.
			for state, d := range sm {
				bm = append(bm, metricsservice.BluemixMetric{
					Name:  strings.Join([]string{metricsPrefix, mval.ClusterName, state, "duration", "min"}, "."),
					Value: d.minTime.Seconds(),
				})
				bm = append(bm, metricsservice.BluemixMetric{
					Name:  strings.Join([]string{metricsPrefix, mval.ClusterName, state, "duration", "max"}, "."),
					Value: d.maxTime.Seconds(),
				})
				bm = append(bm, metricsservice.BluemixMetric{
					Name:  strings.Join([]string{metricsPrefix, mval.ClusterName, state, "duration", "sparse-avg"}, "."),
					Value: d.totalTime.Seconds() / float64(totalWorkers),
				})
			}
		}
	}

	// Generate response time metrics
	sort.Float64s(responseTimes)
	minResponseTime := responseTimes[0]
	maxResponseTime := responseTimes[apiRequestCount-1]
	meanResponseTime := totalResponseTime.Seconds() / float64(apiRequestCount)
	bm = append(bm, metricsservice.BluemixMetric{
		Name:  metricsPrefix + ".Min_Response_Time.min",
		Value: minResponseTime,
	})
	bm = append(bm, metricsservice.BluemixMetric{
		Name:  metricsPrefix + ".Max_Response_Time.max",
		Value: maxResponseTime,
	})
	bm = append(bm, metricsservice.BluemixMetric{
		Name:  metricsPrefix + ".Mean_Response_Time.sparse-avg",
		Value: meanResponseTime,
	})

	p90ResponseTime, err := percentile(responseTimes, 90)
	if err == nil {
		bm = append(bm, metricsservice.BluemixMetric{
			Name:  metricsPrefix + ".P90_Response_Time.sparse-avg",
			Value: p90ResponseTime,
		})
	}

	// Add the action time metrics if it makes sense to do so
	if totalActionTime > 0 {
		sort.Float64s(actionTimes)
		minActionTime := actionTimes[firstAction]
		maxActionTime := actionTimes[apiRequestCount-1]
		meanActionTime := totalActionTime.Seconds() / float64(apiRequestCount-firstAction)
		bm = append(bm, metricsservice.BluemixMetric{
			Name:  metricsPrefix + ".Min_Action_Time.min",
			Value: minActionTime,
		})
		bm = append(bm, metricsservice.BluemixMetric{
			Name:  metricsPrefix + ".Max_Action_Time.max",
			Value: maxActionTime,
		})
		bm = append(bm, metricsservice.BluemixMetric{
			Name:  metricsPrefix + ".Mean_Action_Time.sparse-avg",
			Value: meanActionTime,
		})

		p90ActionTime, errat := percentile(actionTimes, 90)
		if errat == nil {
			bm = append(bm, metricsservice.BluemixMetric{
				Name:  metricsPrefix + ".P90_Action_Time.sparse-avg",
				Value: p90ActionTime,
			})
		}
	}

	if actionFailures > 0 {
		bm = append(bm, metricsservice.BluemixMetric{
			Name:  metricsPrefix + ".Failed_Action_Count",
			Value: actionFailures,
		})
	}

	if backendFailures > 0 {
		bm = append(bm, metricsservice.BluemixMetric{
			Name:  metricsPrefix + ".Failed_Backend_Count",
			Value: backendFailures,
		})
	}

	// Write results to the metrics service
	metricsservice.WriteClusterCreateBluemixMetrics(bm, true, testName, dbKey, clusterNames)
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
