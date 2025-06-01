

package k8stypes

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.ibm.com/alchemy-containers/armada-performance/api/kubernetes/k8s-metrics/config"
	metricsservice "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var pods int32

func podCreated(obj interface{}) {
	pod := obj.(*v1.Pod)
	if pod.Status.Phase != "Failed" && pod.Status.Phase != "Succeeded" {
		pods++
		if config.Verbose {
			log.Printf("Pod Created: %s. Total Pods: %d\n", pod.GetName(), pods)
		}
	}

	if config.Debug {
		fmt.Println(pod.String())
	}
}

func podUpdated(oldObj, newObj interface{}) {
	oldPod := oldObj.(*v1.Pod)
	newPod := newObj.(*v1.Pod)

	if config.Verbose {
		log.Printf("Pod Updated: %s\n", newPod.GetName())
	}

	if config.Debug {
		fmt.Println("OldPod: " + oldPod.String())
		fmt.Println("NewPod: " + newPod.String())
	}
}

func podDeleted(obj interface{}) {
	pod := obj.(*v1.Pod)
	pods--

	log.Printf("Pod Deleted: %s. Target %d. Current %d\n", pod.GetName(), targetReplicas, pods)

	if float64(lastReportedReplicas)-float64(pods) >= (float64(initialReplicas-targetReplicas) * 0.1) {
		lastReportedReplicas = pods
		pcntComplete := int(float64(initialReplicas-lastReportedReplicas) / float64(initialReplicas-targetReplicas) * 100.0)
		pcntCompleteStr := strconv.Itoa(pcntComplete) + "_Pcnt"
		bm = append(bm,
			metricsservice.BluemixMetric{
				Name:      strings.Join([]string{"k8s", metricName, pod.GetNamespace(), pcntCompleteStr, "sparse-avg"}, "."),
				Timestamp: time.Now().Unix(),
				Value:     pods,
			},
		)
	}

	if config.Debug {
		fmt.Println(pod.String())
	}

	if pods == targetReplicas {
		log.Printf("Pod Deletion: %s, Complete\n", pod.GetNamespace())

		// Generate summary metric
		testDuration := bm[len(bm)-1].Timestamp - bm[0].Timestamp
		bm = append(bm,
			metricsservice.BluemixMetric{
				Name:      strings.Join([]string{"k8s", metricName, pod.GetNamespace(), "duration", "sparse-avg"}, "."),
				Timestamp: time.Now().Unix(),
				Value:     testDuration,
			},
		)
		if testRunning {
			log.Printf("Test \"%s\" completed.\n", testName)
			testRunning = false

			if config.Metrics {
				metricsservice.WriteBluemixMetrics(bm, true, "", "")
			}
			log.Println(bm)
			bm = nil
		}
	}
}

// WatchPods sets up a watcher for k8s pod events
func WatchPods(store cache.Store, kubeConfig *kubernetes.Clientset, namespace string) cache.Store {
	//Define what we want to look for (Pods)
	watchlist := cache.NewListWatchFromClient(
		kubeConfig.CoreV1().RESTClient(),
		"pods",
		namespace,
		fields.Everything())

	//Setup an informer to call functions when the watchlist changes
	eStore, eController := cache.NewInformer(
		watchlist,
		&v1.Pod{},
		0*time.Second,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    podCreated,
			UpdateFunc: podUpdated,
			DeleteFunc: podDeleted,
		},
	)

	//Run the controller as a goroutine
	go eController.Run(wait.NeverStop)
	return eStore
}
