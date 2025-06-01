/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2018, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

var (
	// Prometheus configuration data
	Prometheus PromConfig

	// CarrierKubeconfig is the location of the kubeconfig file for carrier access (required to access prometheus pod)
	CarrierKubeconfig string

	// Verbose output required?
	Verbose bool

	// Publish data to metrics service
	Publish bool
)

type promMetricsConfig struct {
	Name  string `toml:"name"`
	Query string `toml:"query"`
}

// PromConfig defines the Prometheus metrics to be collected
type PromConfig struct {
	Environment string                         `toml:"environment"`
	Carrier     string                         `toml:"carrier"`
	Port        int                            `toml:"port"`
	Devices     string                         `toml:"devices"`
	Metrics     map[string][]promMetricsConfig `toml:"metrics"`
}

// GetConfigPath returns the path of toml config file
func GetConfigPath() string {
	configPath := os.Getenv("CARRIER_METRICS_CONFIG_PATH")
	if configPath != "" {
		return configPath
	}
	goPath := os.Getenv("GOPATH")
	srcPath := filepath.Join("src", "github.ibm.com", "alchemy-containers", "armada-performance", "metrics", "carrier")
	return filepath.Join(goPath, srcPath, "config")
}

// ParseConfig parses the toml configuration file with results stored in the supplied config argument
func ParseConfig(filePath string, conf interface{}) {
	if _, err := toml.DecodeFile(filePath, conf); err != nil {
		log.Fatalf("Error parsing config file : %s\n", err.Error())
	}
}
