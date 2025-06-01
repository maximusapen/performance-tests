/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
	"flag"
	"log"
	"strings"

	"github.ibm.com/alchemy-containers/armada-performance/persistent-storage/snapshotStorage/config"
	"github.ibm.com/alchemy-containers/armada-performance/persistent-storage/snapshotStorage/pod"
	volumesnapshot "github.ibm.com/alchemy-containers/armada-performance/persistent-storage/snapshotStorage/volumeSnapshot"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ev "github.com/kubernetes-csi/external-snapshotter/client/v6/informers/externalversions"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

type resources []string

func (i *resources) String() string {
	return "resource"
}

func (i *resources) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var (
	rtm resources

	kubeClientset *kubernetes.Clientset
)

func main() {
	flag.StringVar(&config.KubeConfig, "kubecfg", "", "Location of the kubernetes client configuration file. Default is to use $KUBECONFIG.")
	flag.StringVar(&config.Namespace, "namespace", metav1.NamespaceDefault, "Kubernetes namespace used for running tests.")
	flag.BoolVar(&config.Verbose, "verbose", false, "verbose logging output")
	flag.BoolVar(&config.Debug, "debug", false, "debug logging output")
	flag.BoolVar(&config.Metrics, "metrics", false, "Send results/metrics to the armada performance metrics service. Defaults to no metrics")
	flag.Var(&rtm, "resource", "Resource type to monitor, e.g. pod, volumeSnapshot")
	flag.Parse()

	for _, r := range rtm {
		switch strings.ToLower(r) {
		case "volumesnapshots":
			//factory := ev.NewSharedInformerFactory(config.GetExternalSnapshotterKubeClient(), 0)
			factory := ev.NewSharedInformerFactoryWithOptions(config.GetExternalSnapshotterKubeClient(), 0, ev.WithNamespace(config.Namespace))
			volumesnapshot.WatchVolumeSnapshots(factory)
		case "pods":
			factory := informers.NewSharedInformerFactoryWithOptions(config.GetKubeClient(), 0, informers.WithNamespace(config.Namespace))
			pod.WatchPods(factory)
		default:
			log.Fatalf("Unrecognized resource type : %s\n", r)
		}
	}
	//informer.GetStore()

	//Keep alive
	select {}
}
