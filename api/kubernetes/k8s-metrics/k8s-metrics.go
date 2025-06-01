
package main

import (
	"flag"
	"fmt"
	"os"

	"github.ibm.com/alchemy-containers/armada-performance/api/kubernetes/k8s-metrics/config"
	types "github.ibm.com/alchemy-containers/armada-performance/api/kubernetes/k8s-metrics/types"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	kubeClientset *kubernetes.Clientset
	kubeConfig    string
	namespace     string
)

func main() {
	flag.StringVar(&kubeConfig, "kubecfg", "", "Location of the kubernetes client configuration file. Default is to use $KUBECONFIG.")
	flag.StringVar(&namespace, "namespace", v1.NamespaceDefault, "Kubernetes namespace to be monitored")
	flag.BoolVar(&config.Verbose, "verbose", false, "verbose logging output")
	flag.BoolVar(&config.Debug, "debug", false, "debug logging output")
	flag.BoolVar(&config.Metrics, "metrics", false, "Send results/metrics to Armada Metrics service. Defaults to no metrics")
	flag.Parse()

	//Create a cache to store Deployments
	var deploymentStore, podStore cache.Store

	//Watch for Deployments and Pods
	kubeClientConfig := getKubeClientSet()

	_ = types.WatchDeployments(deploymentStore, kubeClientConfig, namespace)
	_ = types.WatchPods(podStore, kubeClientConfig, namespace)

	//Keep alive
	select {}
}

// getKubeClientSet loads the k8s config for the cluster
func getKubeClientSet() *kubernetes.Clientset {
	if kubeConfig == "" {
		kubeConfig = os.Getenv("KUBECONFIG")
	}

	if kubeConfig != "" {
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		clientset, _ := kubernetes.NewForConfig(config)
		kubeClientset = clientset
	} else {
		fmt.Fprintln(os.Stderr, "Error: Kube configuration isn't defined. Set KUBECONFIG or specify as parameter")
		os.Exit(1)
	}

	return kubeClientset
}
