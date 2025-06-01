/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2021, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package influxdata

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.ibm.com/alchemy-containers/armada-performance/metrics/alerting/config"

	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	client "github.com/influxdata/influxdb1-client/v2"
)

const dbPwdEnvVar = "METRICS_DB_PWD"

// InfluxClient holds data for accessing test results from Influx
type InfluxClient struct {
	config config.InfluxDB
	client *client.Client
}

// TestResult holds a single test result value
type TestResult struct {
	Timestamp int64
	Val       float64
}

// TestResults holds historical and current test result data for a single test
type TestResults struct {
	Current    []TestResult
	Historical []TestResult
}

// NewInfluxClient returns a client for accessing an InfluxDB instance
func NewInfluxClient(cfg config.InfluxDB) InfluxClient {
	influx := InfluxClient{config: cfg}

	dbKey := os.Getenv(dbPwdEnvVar)
	if len(dbKey) == 0 {
		log.Printf("WARNING: Influx DB password not provided. Check '%s' environment variable", dbPwdEnvVar)
	}

	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     "http://" + net.JoinHostPort(influx.config.Host, strconv.Itoa(influx.config.Port)),
		Username: influx.config.Username,
		Password: dbKey, // pragma: allowlist secret
		Timeout:  time.Duration(influx.config.Timeout) * time.Second,
	})
	if err != nil {
		log.Fatalf("Error creating InfluxDB Client: %s", err.Error())
	}
	influx.client = &c

	return influx
}

// GetTestData returns test reuslts data from InfluxDB
func (ic InfluxClient) GetTestData(carrier string, test string, kubeVersion string, machineType string, operatingSystem string, item string, limit config.History) TestResults {
	var limitCount string
	var limitDays string

	if limit.Count > 0 {
		limitCount = fmt.Sprintf(" LIMIT %d", limit.Count)
	}

	if limit.Days > 0 {
		limitDays = fmt.Sprintf(" AND time > now() - %dd", limit.Days)
	}

	if len(kubeVersion) > 0 {
		kubeVersion = fmt.Sprintf(" AND (KubeVersion = '%s' or KubeVersion = '')", kubeVersion)
	}

	if len(machineType) > 0 {
		machineType = strings.ReplaceAll(machineType, ".", "_")
		machineType = fmt.Sprintf(" AND (MachineType = '%s' or MachineType = '')", machineType)
	}

	if len(operatingSystem) > 0 {
		operatingSystem = fmt.Sprintf(" AND (OperatingSystem = '%s' or OperatingSystem = '')", operatingSystem)
	}

	// Example query:
	//   SELECT nodes5_replicas3_threads_20_singlezone_throughput FROM httpnodeport
	//   WHERE MetricName = 'nodes5_replicas3_threads_20_singlezone_throughput'
	//   AND CarrierName = 'carrier3_stage'
	//   AND MachineType = 'b1_4x16'
	//   AND (KubeVersion = '1_23' or KubeVersion = '')
	//   AND (OperatingSystem = 'RHEL_7' or OperatingSystem = '')
	//   AND time > now() - 5d
	//   ORDER BY time DESC LIMIT 5
	c := *ic.client
	cmd := fmt.Sprintf("SELECT %s FROM %s where CarrierName = '%s'%s%s%s %s ORDER BY time DESC%s", item, test, carrier, kubeVersion, machineType, operatingSystem, limitDays, limitCount)

	if config.GetConfig().Options.Debug {
		fmt.Print(config.ColourFaint)
		fmt.Println(cmd)
		fmt.Print(config.ColourReset)
	}
	q := client.NewQuery(cmd, ic.config.Database, "s")
	if response, err := c.Query(q); err == nil && response.Error() == nil {
		if config.GetConfig().Options.Debug {
			fmt.Print(config.ColourFaint)
			fmt.Println(response)
			fmt.Print(config.ColourReset)
		}
		if len(response.Results[0].Series) > 0 {
			inflxuResults := response.Results[0].Series[0].Values

			results := TestResults{
				Current:    make([]TestResult, 0, limit.Current),
				Historical: make([]TestResult, 0, limit.Count),
			}

			for i, v := range inflxuResults {
				ts, err := v[0].(json.Number).Int64()
				if err != nil {
					log.Fatalf("Unexpected timestamp value : %s\n", err)
				}
				v, err := v[1].(json.Number).Float64()
				if err != nil {
					log.Fatalf("Unexpected data value : %s\n", err)
				}

				if i < limit.Current {
					results.Current = append(results.Current, TestResult{Timestamp: ts, Val: v})
				} else {
					results.Historical = append(results.Historical, TestResult{Timestamp: ts, Val: v})
				}
			}
			return results
		}
	} else {
		if err != nil {
			log.Fatalf("Error running InfluxDB query: %v", err.Error())
		}
	}

	return TestResults{
		Current:    make([]TestResult, 0),
		Historical: make([]TestResult, 0),
	}
}
