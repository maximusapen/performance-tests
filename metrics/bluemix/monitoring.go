/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package metricsservice

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.ibm.com/alchemy-containers/armada-performance/tools/crypto/utils"
)

func getGoPath() string {
	if goPath := os.Getenv(strings.ToUpper("GOPATH")); goPath != "" {
		return goPath
	}
	return ""
}

func getPerfRepoPath() string {
	configPath := os.Getenv("ARMADA_PERF_REPO_PATH")
	if configPath != "" {
		return configPath
	}
	goPath := getGoPath()
	srcPath := filepath.Join("src", "github.ibm.com", "alchemy-containers", "armada-performance")
	return filepath.Join(goPath, srcPath)
}

// SetConfigPath overrides the default location for configuration files
func SetConfigPath(path string) {
	configPath = path
}

// metricsConfig - note Influx DB password deliberately not read from file for security reasons.
type metricsConfig struct {
	Root             string `toml:"root"`
	Scheme           string `toml:"scheme"`
	Host             string `toml:"host"`
	Path             string `toml:"path"`
	Verbose          bool   `toml:"verbose"`
	RateLimit        int    `toml:"ratelimit"` // Requests per second
	InfluxdbHost     string `toml:"influxdbHost"`
	InfluxdbPort     string `toml:"influxdbPort"`
	InfluxdbName     string `toml:"influxdbName"`
	InfluxdbUser     string `toml:"influxdbUser"`
	InfluxdbPassword string `toml:"influxdbPassword"`
	InfluxdbVerbose  bool   `toml:"influxdbVerbose"`
}

// ServiceConfig contains the metrics configuration data
type ServiceConfig struct {
	Metrics *metricsConfig
}

// BluemixMetric defines the structure required by the Bluemix Metrics service
type BluemixMetric struct {
	Name      string      `json:"name"`
	Timestamp int64       `json:"timestamp,omitempty"`
	Value     interface{} `json:"value"`
}

var metricsCfg ServiceConfig
var configPath string
var clusterNames []string
var carrierMetrics = false

// ReadMetricsTomlFile reads the the metrics.toml file
func ReadMetricsTomlFile() (ServiceConfig, bool) {
	var localMetricsCfg ServiceConfig
	success := false
	path := filepath.Join(getPerfRepoPath(), "metrics", "bluemix", "metrics.toml")
	if len(configPath) > 0 {
		path = filepath.Join(configPath, "metrics.toml")
	}

	// When Build and Copy repo job is running it can fail to read the file, so put in some retries
	retries := 0
	maxRetries := 5
	for !success && (retries < maxRetries) {
		if _, err := toml.DecodeFile(path, &localMetricsCfg); err != nil {
			log.Println("Error parsing 'metrics.toml' config file. No metrics data will be available\n" + err.Error())
			success = false
			retries++
			time.Sleep(30 * time.Second)
		} else {
			success = true
		}
	}
	if success {
		if len(localMetricsCfg.Metrics.Host) == 0 || len(localMetricsCfg.Metrics.Scheme) == 0 || len(localMetricsCfg.Metrics.Path) == 0 {
			// Invalid metrics config specified in 'metrics.toml'
			log.Println("Invalid metrics configuration information specified. No metrics data will be processed.")
			log.Println(*localMetricsCfg.Metrics)
			success = false
		}

	}
	return localMetricsCfg, success
}

// WriteClusterCreateBluemixMetrics is a special version of WriteBluemix Metrics. It is used by createCluster which needs to pass in an extra parameter i.e. the cluster name.
// The clusterName can then be added as a metric tag and removed from the metric name
func WriteClusterCreateBluemixMetrics(metrics []BluemixMetric, addPrefix bool, testName string, dbKey string, theClusterNames []string) {
	clusterNames = theClusterNames
	WriteBluemixMetrics(metrics, addPrefix, testName, dbKey)
}

// WriteCarrierBluemixMetrics is a special version of WriteBluemix Metrics. It is used by Carrier metrics code and allows us to avoid writing the large number of Carrier metrics to the logs
// The clusterName can then be added as a metric tag and removed from the metric name
func WriteCarrierBluemixMetrics(metrics []BluemixMetric, addPrefix bool, testName string, dbKey string) {
	carrierMetrics = true
	WriteBluemixMetrics(metrics, addPrefix, testName, dbKey)
	carrierMetrics = false
}

// WriteBluemixMetrics sends the supplied metrics to the Bluemix metrics service
// The InfluxDB password will be searched for:
// 1. Passed in as parameter by caller
// 2. From METRICS_DB_KEY Environment variable
// 3. From metrics.toml
func WriteBluemixMetrics(metrics []BluemixMetric, addPrefix bool, testName string, dbKey string) {
	// Get metrics configuration data
	var ok bool
	if metricsCfg, ok = ReadMetricsTomlFile(); !ok {
		log.Println("Unable to read metrics toml file - metrics will not be published")
		return
	}

	// Get any user defined prefix and/or kubernetes server version info.
	ump := os.Getenv("METRICS_PREFIX")
	k8sVersion := os.Getenv("K8S_SERVER_VERSION")
	K8sVersionShort := k8sVersion
	splitK8Ver := strings.Split(strings.Split(k8sVersion, "_")[0], ".")
	if len(splitK8Ver) > 1 {
		K8sVersionShort = splitK8Ver[0] + "_" + splitK8Ver[1]
	}

	// Get testname from env var if not set
	if len(testName) == 0 {
		testName = os.Getenv("TEST_NAME")
	}
	testNameReplacer := strings.NewReplacer(" ", "")
	correctedTestName := strings.ToLower(testNameReplacer.Replace(testName))

	// See if DB Key is set as Env Variable
	if len(dbKey) == 0 {
		dbKey = os.Getenv("METRICS_DB_KEY")
		// See if DB Key is is in metrics.toml - doesn't matter if this is empty - we will just write the results as a file
		if len(dbKey) == 0 {
			// Need to decrypt the api keys - but only if the encryption key is set
			// The encryption key needs to be setup by the caller
			// If it isn't set that's fine, the DB Key will be empty, so metrics will be written to file
			encryptionKey := os.Getenv(utils.KeyEnvVar)
			if len(encryptionKey) > 0 {
				var err error
				metricsCfg.Metrics.InfluxdbPassword, err = utils.Decrypt(metricsCfg.Metrics.InfluxdbPassword)
				if err != nil {
					log.Printf("Error decrypting InfluxDB Pasword from metrics.toml, metrics will be written to a file : %s\n", err.Error()) // pragma: allowlist secret
					dbKey = ""
				} else {
					dbKey = metricsCfg.Metrics.InfluxdbPassword
				}
			}
		}
	}

	// Send the metrics to Influxdb. Must send them before we add the prefix and k8s version - which are better passed as tags
	WriteInfluxdbData(metrics, correctedTestName, ump, metricsCfg.Metrics.Root, K8sVersionShort, metricsCfg.Metrics.InfluxdbHost, metricsCfg.Metrics.InfluxdbPort, metricsCfg.Metrics.InfluxdbName, metricsCfg.Metrics.InfluxdbUser, dbKey, metricsCfg.Metrics.InfluxdbVerbose, clusterNames, carrierMetrics)
}
