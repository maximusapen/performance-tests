/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package volumesnapshot

import (
	"log"
	"strings"
	"time"

	"github.ibm.com/alchemy-containers/armada-performance/persistent-storage/snapshotStorage/config"
	"github.ibm.com/alchemy-containers/armada-performance/persistent-storage/snapshotStorage/pod"

	metricsservice "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"

	crdv1 "github.com/kubernetes-csi/external-snapshotter/client/v6/apis/volumesnapshot/v1"
	esevv6 "github.com/kubernetes-csi/external-snapshotter/client/v6/informers/externalversions"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
)

var startTime time.Time

// WatchVolumeSnapshots monitors volumeSnapshot events
func WatchVolumeSnapshots(factory esevv6.SharedInformerFactory) {
	// Get the informer for the right resource, in this case a Pod
	informer := factory.Snapshot().V1().VolumeSnapshots().Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// voluneSnapshot created
		AddFunc: func(obj interface{}) {
			startTime = time.Now()

			if config.Debug {
				vs := obj.(*crdv1.VolumeSnapshot)
				log.Printf("VolumeSnapshot creation - '%s'\n", vs.GetName())
			}
		},
		// voluneSnapshot updated
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldVS := oldObj.(*crdv1.VolumeSnapshot)
			newVS := newObj.(*crdv1.VolumeSnapshot)
			if oldVS.Status != nil && newVS.Status != nil {
				if !*oldVS.Status.ReadyToUse && *newVS.Status.ReadyToUse {
					endTime := time.Now()
					creationTime := endTime.Sub(startTime)

					if config.Verbose {
						log.Printf("Time to create volumeSnapshot : %.1fs\n", creationTime.Seconds())
					}

					// Get the volume capacity and data size which should have been set as labels on the setup pod
					volumeSize, volumeDataSize := "unknown", "unknown"
					setupPod := pod.GetPodsByLabel("use", "snapshot-storage-setup")
					if len(setupPod) == 1 {
						setupPodLabels := setupPod[0].GetLabels()
						volumeSize = setupPodLabels["volume-size"]
						volumeDataSize = setupPodLabels["volume-data-size"]
					}

					// Generate metrics
					var bm []metricsservice.BluemixMetric
					bm = append(bm,
						metricsservice.BluemixMetric{
							Name:      strings.Join([]string{config.MetricsRootName, volumeSize, "backup", volumeDataSize, "sparse-avg"}, "."),
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
		},
		// voluneSnapshot deleted
		DeleteFunc: func(obj interface{}) {
			if config.Debug {
				vs := obj.(*crdv1.VolumeSnapshot)
				log.Printf("VolumeSnapshot deletion - '%s'\n", vs.GetName())
			}
		},
	})

	log.Println("Monitoring VolumeSnapshot events")
	go informer.Run(wait.NeverStop)
}
