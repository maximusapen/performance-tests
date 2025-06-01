/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package config

import (
	"log"
	"os"
	"strings"

	esclientset "github.com/kubernetes-csi/external-snapshotter/client/v6/clientset/versioned"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Metrics holds command line option to indicate whether metrics should be sent to metrics database
var Metrics bool

// Verbose holds command line option to indicate whether additional logging is required
var Verbose bool

// Debug holds command line option to indicate whether detailed logging is required
var Debug bool

// KubeConfig holds the location of the kube configuration file
var KubeConfig string

// Namespace used for running tests
var Namespace string

var kubeClient *kubernetes.Clientset
var esKubeClient *esclientset.Clientset

// MetricsRootName is the root identifier applied to all metrics
var MetricsRootName = strings.Join([]string{"block_storage", "volume_snapshot"}, ".")

// GetKubeClient creates the kube client
func GetKubeClient() kubernetes.Interface {
	if kubeClient == nil {
		if KubeConfig == "" {
			KubeConfig = os.Getenv("KUBECONFIG")
		}

		if KubeConfig != "" {
			config, err := clientcmd.BuildConfigFromFlags("", KubeConfig)
			if err != nil {
				log.Fatalln(err)
			}

			client, err := kubernetes.NewForConfig(config)
			if err != nil {
				log.Fatalln(err)
			}

			return client
		}

		log.Fatalln("Error: Kube configuration isn't defined. Set KUBECONFIG or specify --kubecfg parameter")
	}
	return kubeClient
}

// GetExternalSnapshotterKubeClient creates the csi external-snapshotter kube client
func GetExternalSnapshotterKubeClient() *esclientset.Clientset {
	if esKubeClient == nil {
		if KubeConfig == "" {
			KubeConfig = os.Getenv("KUBECONFIG")
		}

		if KubeConfig != "" {
			config, err := clientcmd.BuildConfigFromFlags("", KubeConfig)
			if err != nil {
				log.Fatalln(err)
			}

			client, err := esclientset.NewForConfig(config)
			if err != nil {
				log.Fatalln(err)
			}

			return client
		}
		log.Fatalln("Error: Kube configuration isn't defined. Set KUBECONFIG or specify --kubecfg parameter")
	}
	return esKubeClient
}
