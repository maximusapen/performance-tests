/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2022, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package pod

import (
	"context"
	"log"
	"strings"
	"time"

	metricsservice "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"
	"github.ibm.com/alchemy-containers/armada-performance/persistent-storage/snapshotStorage/config"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

var startTime map[string]time.Time

// GetPodsByLabel returns a list of pods in the configured namesapce with the matching label
func GetPodsByLabel(ln, lv string) []v1.Pod {
	lr, err := labels.NewRequirement(ln, selection.Equals, []string{lv})
	if err != nil {
		log.Fatalln(err)
	}

	selector := labels.NewSelector().Add(*lr)
	podListOptions := metav1.ListOptions{
		LabelSelector: selector.String(),
	}

	pods, err := config.GetKubeClient().CoreV1().Pods(config.Namespace).List(context.TODO(), podListOptions)
	if err != nil {
		log.Fatalln(err)
	}

	return pods.Items
}

// WatchPods monitors pod events
func WatchPods(factory informers.SharedInformerFactory) {
	// Get the informer for the right resource, in this case a Pod
	informer := factory.Core().V1().Pods().Informer()

	startTime = make(map[string]time.Time)

	// This is the part where your custom code gets triggered based on the
	// event that the shared informer catches
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// pod created
		AddFunc: func(obj interface{}) {
			pod := obj.(*v1.Pod)
			startTime[pod.GetName()] = time.Now()

			if config.Debug {
				log.Printf("Pod creation event - '%s'\n", pod.GetName())
			}
		},

		// pod updated
		UpdateFunc: func(oldObj interface{}, newObj interface{}) {
			oldPod := oldObj.(*v1.Pod)
			newPod := newObj.(*v1.Pod)

			for c := range oldPod.Status.ContainerStatuses {
				if !oldPod.Status.ContainerStatuses[c].Ready && newPod.Status.ContainerStatuses[c].Ready {
					endTime := time.Now()
					creationTime := endTime.Sub(startTime[newPod.GetName()])

					if config.Verbose {
						log.Printf("Time to create pod '%s' with pvc : %.1fs\n", newPod.GetName(), creationTime.Seconds())
					}

					for _, v := range newPod.Spec.Volumes {
						if pvc := v.PersistentVolumeClaim; pvc != nil {
							c, err := config.GetKubeClient().CoreV1().PersistentVolumeClaims(config.Namespace).Get(context.TODO(), pvc.ClaimName, metav1.GetOptions{})
							if err != nil {
								log.Printf("Failed to find pvc '%s'\n", pvc.ClaimName)
								continue
							}
							ds := c.Spec.DataSource
							if ds != nil && ds.Kind == "VolumeSnapshot" {
								// Get the volume capacity and data size which should have been set as labels on the setup pod
								volumeSize, volumeDataSize := "unknown", "unknown"
								setupPod := GetPodsByLabel("use", "snapshot-storage-setup")
								if len(setupPod) == 1 {
									setupPodLabels := setupPod[0].GetLabels()
									volumeSize = setupPodLabels["volume-size"]
									volumeDataSize = setupPodLabels["volume-data-size"]
								}

								// Generate metrics
								var bm []metricsservice.BluemixMetric
								bm = append(bm,
									metricsservice.BluemixMetric{
										Name:      strings.Join([]string{config.MetricsRootName, volumeSize, "restore", volumeDataSize, "sparse-avg"}, "."),
										Timestamp: time.Now().Unix(),
										Value:     creationTime.Seconds(),
									},
								)

								if config.Metrics {
									metricsservice.WriteBluemixMetrics(bm, true, "", "")
								}
								log.Println(bm)
							}
						}
					}
				}
			}
		},

		// pod deleted
		DeleteFunc: func(obj interface{}) {
			if config.Debug {
				pod := obj.(*v1.Pod)
				log.Printf("Pod deletion event - '%s'\n", pod.GetName())
			}
		},
	})

	log.Println("Monitoring Pod events")
	go informer.Run(wait.NeverStop)
}
