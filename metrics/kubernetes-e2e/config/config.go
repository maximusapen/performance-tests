/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2021 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package config

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// TestcasesFile holds the filename of the metrics configuration file as supplied by the user.
var TestcasesFile string

// Verbose controls whether verbose output to stdout should be generated
var Verbose bool

// ResultsDir holds the name of the directory contianing Kubernetes E2E Results files
var ResultsDir string

// IBMMetrics controls whether results are sent to the IBM Cloud Metrics/Monitoring service (Grafana)
var IBMMetrics bool

func getEnv(key string) string {
	return os.Getenv(strings.ToUpper(key))
}

// GetGoPath returns gopath if defined
func GetGoPath() string {
	if goPath := getEnv("GOPATH"); goPath != "" {
		return goPath
	}
	return ""
}

// GetConfigPath returns path of toml config file
func GetConfigPath() string {
	goPath := GetGoPath()
	srcPath := filepath.Join("src", "github.ibm.com", "alchemy-containers", "armada-performance", "metrics", "kubernetes-e2e")
	return filepath.Join(goPath, srcPath, "config")
}

// GetClusterNodeCount uses the Kubernetes api to obtain get the number of nodes in a cluster
func GetClusterNodeCount(kubeconfig *string) (int, int, bool) {
	var nodeCount, schedulableNodeCount int
	var hollowNodes bool
	if kubeconfig != nil {
		// use the current context in kubeconfig
		kubeConfig, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			log.Printf("Unable to determine number of nodes in cluster. %s\n", err.Error())
			return 0, 0, false
		}

		// create the clientset
		clientset, err := kubernetes.NewForConfig(kubeConfig)
		if err != nil {
			log.Printf("Unable to determine number of nodes in cluster. %s\n", err.Error())
			return 0, 0, false
		}

		// Get the number of nodes in the cluster
		nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Printf("Unable to determine number of nodes in cluster. %s\n", err.Error())
			return 0, 0, false
		}

		nodeCount = len(nodes.Items)
		for _, n := range nodes.Items {
			if !n.Spec.Unschedulable {
				schedulableNodeCount++
			}
			if !hollowNodes && strings.Index(n.Name, "hollow-node") >= 0 {
				hollowNodes = true
			}
		}
	}
	return nodeCount, schedulableNodeCount, hollowNodes
}

// Config defines the structure of the toml config file that defines which Kubernetes metrics to gather from results
type Config struct {
	Load    loadConfig    `toml:"load"`
	Density densityConfig `toml:"density"`
}

type loadConfig struct {
	MetricsForE2E            MetricsForE2E                         `toml:"MetricsForE2E"`
	APIResponsivenessOverall map[string][]APIResponsivenessOverall `toml:"APIResponsivenessOverall"`
	APIResponsiveness        map[string][]APIResponsiveness        `toml:"APIResponsiveness"`
	PodStartupLatency        map[string][]APIResponsiveness        `toml:"PodStartupLatency"`
	TestPhaseTimer           struct{ Report bool }                 `toml:"TestPhaseTimer"`
}

// MetricsForE2E defines the structure of Kubernetes e2e metrics test results
type MetricsForE2E struct {
	Latency map[string][]apiServerRequestLatencies `toml:"APIServer_Request_Latencies"`
}

// APIResponsivenessOverall defines the structure of Kubernetes API ResponsivenessOverall prometheus test results
type APIResponsivenessOverall struct {
	Verb        string `toml:"verb"`
	SubResource string `toml:"subResource"`
}

// APIResponsiveness defines the structure of Kubernetes API Responsiveness test results
type APIResponsiveness struct {
	Verb        string   `toml:"Verb"`
	SubResource string   `toml:"SubResource"`
	Data        []string `toml:"data"`
}

type apiServerRequestLatencies struct {
	Resource    string `toml:"Resource"`
	SubResource string `toml:"SubResource"`
	Verb        string `toml:"Verb"`
	Quantile    string `toml:"Quantile"`
}

type densityConfig struct {
	MetricsForE2E     MetricsForE2E                  `toml:"MetricsForE2E"`
	APIResponsiveness map[string][]APIResponsiveness `toml:"APIResponsiveness"`
	PodStartupLatency map[string][]APIResponsiveness `toml:"PodStartupLatency"`
	SchedulingLatency densitySchedulingLatency       `toml:"SchedulingLatency"`
	TestPhaseTimer    struct{ Report bool }          `toml:"TestPhaseTimer"`
}

type densityPodStartupLatency struct {
	Data []string `toml:"data"`
}

type densitySchedulingLatency struct{} //Implement when tests are supported on Armada

type testPhaseTimerConfig struct{}

// ParseConfig parses the toml configuration file with results stored in the supplied config argument
func ParseConfig(filePath string, conf interface{}) {
	if _, err := toml.DecodeFile(filePath, conf); err != nil {
		log.Fatalf("error parsing config file : %s\n", err.Error())
	}
}
