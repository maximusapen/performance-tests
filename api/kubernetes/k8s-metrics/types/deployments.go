/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package k8stypes

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.ibm.com/alchemy-containers/armada-performance/api/kubernetes/k8s-metrics/config"
	metricsservice "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var (
	testRunning                                           bool
	testName                                              string
	metricName                                            string
	replicaDiff                                           int32
	initialReplicas, targetReplicas, lastReportedReplicas int32

	// Holds our metrics to be sent to Bluemix metric service
	bm []metricsservice.BluemixMetric
)

func deploymentCreated(obj interface{}) {
	deployment := obj.(*v1.Deployment)

	log.Printf("Deployment Created: %s, Desired Replicas: %d, Initial Available Replicas: %d\n", deployment.GetName(), *deployment.Spec.Replicas, deployment.Status.AvailableReplicas)
	if config.Debug {
		fmt.Println(deployment.String())
	}
}

func deploymentUpdated(oldObj, newObj interface{}) {
	oldDeployment := oldObj.(*v1.Deployment)
	newDeployment := newObj.(*v1.Deployment)

	// Use 0 replicas as an indication we're done
	if *newDeployment.Spec.Replicas == 0 {
		log.Println("Completion event (zero replicas) received. Exiting")
		os.Exit(0)
	}

	if !testRunning {
		log.Printf("Deployment Updated: %s, Desired Replicas: %d, Available Replicas: %d\n", newDeployment.GetName(), *newDeployment.Spec.Replicas, newDeployment.Status.AvailableReplicas)
		replicas := fmt.Sprintf("%d-%d", *oldDeployment.Spec.Replicas, *newDeployment.Spec.Replicas)
		if *newDeployment.Spec.Replicas > *oldDeployment.Spec.Replicas {
			testName = config.ScalingUpName
			metricName = strings.Join([]string{config.ScalingUpName, replicas, "available_replicas"}, ".")
		} else if *newDeployment.Spec.Replicas < *oldDeployment.Spec.Replicas {
			testName = config.ScalingDownName
			metricName = strings.Join([]string{config.ScalingDownName, replicas, "not_deleted"}, ".")
		} else {
			return
		}

		initialReplicas = *oldDeployment.Spec.Replicas
		targetReplicas = *newDeployment.Spec.Replicas
		lastReportedReplicas = initialReplicas

		bm = append(bm,
			metricsservice.BluemixMetric{
				Name:      strings.Join([]string{"k8s", metricName, newDeployment.GetNamespace(), "0_Pcnt", "sparse-avg"}, "."),
				Timestamp: time.Now().Unix(),
				Value:     lastReportedReplicas,
			},
		)

		log.Printf("Test \"%s\" starting.\n", testName)
		testRunning = true
	}

	if config.Debug {
		fmt.Println("OldDeployment: " + oldDeployment.String())
		fmt.Println("NewDeployment: " + newDeployment.String())
	}

	if newDeployment.Status.Replicas != oldDeployment.Status.Replicas {
		if replicaDiff = newDeployment.Status.Replicas - oldDeployment.Status.Replicas; replicaDiff < 0 {
			replicaDiff = -replicaDiff
		}
	}
	if newDeployment.Status.AvailableReplicas != oldDeployment.Status.AvailableReplicas {
		log.Printf("Deployment: %s, Available Replicas: %d\n", newDeployment.GetName(), newDeployment.Status.AvailableReplicas)

		if float64(newDeployment.Status.AvailableReplicas) >= float64(lastReportedReplicas)+(float64(replicaDiff)*0.1) {
			lastReportedReplicas = newDeployment.Status.AvailableReplicas
			pcntComplete := int(float64(lastReportedReplicas-initialReplicas) / float64(replicaDiff) * 100.0)
			pcntCompleteStr := strconv.Itoa(pcntComplete) + "_Pcnt"

			bm = append(bm,
				metricsservice.BluemixMetric{
					Name:      strings.Join([]string{"k8s", metricName, newDeployment.GetNamespace(), pcntCompleteStr, "sparse-avg"}, "."),
					Timestamp: time.Now().Unix(),
					Value:     lastReportedReplicas,
				},
			)
		}
		if newDeployment.Status.UnavailableReplicas == 0 &&
			newDeployment.Status.Conditions[0].Status == "True" {
			log.Printf("Deployment: %s, Complete\n", newDeployment.GetName())

			if testRunning {
				// Scaling up test ?
				if targetReplicas > initialReplicas {
					log.Printf("Test \"%s\" completed.\n", testName)
					testRunning = false

					// Generate summary metric
					testDuration := bm[len(bm)-1].Timestamp - bm[0].Timestamp
					bm = append(bm,
						metricsservice.BluemixMetric{
							Name:      strings.Join([]string{"k8s", metricName, newDeployment.GetNamespace(), "duration", "sparse-avg"}, "."),
							Timestamp: time.Now().Unix(),
							Value:     testDuration,
						},
					)
					if config.Metrics {
						metricsservice.WriteBluemixMetrics(bm, true, "", "")
					}
					fmt.Println(bm)
					bm = nil
				}
			}
		}
	}
}

func deploymentDeleted(obj interface{}) {
	deployment := obj.(*v1.Deployment)

	log.Printf("Deployment Deleted: %s\n", deployment.GetName())
	if config.Debug {
		fmt.Println(deployment.String())
	}
}

// WatchDeployments sets up a watcher for k8s deployment events
func WatchDeployments(store cache.Store, kubeConfig *kubernetes.Clientset, namespace string) cache.Store {
	//Define what we want to look for (Deployments)
	watchlist := cache.NewListWatchFromClient(
		kubeConfig.AppsV1().RESTClient(),
		"deployments",
		namespace,
		fields.Everything())

	//Setup an informer to call functions when the watchlist changes
	eStore, eController := cache.NewInformer(
		watchlist,
		&v1.Deployment{},
		0*time.Second,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    deploymentCreated,
			UpdateFunc: deploymentUpdated,
			DeleteFunc: deploymentDeleted,
		},
	)

	//Run the controller as a goroutine
	go eController.Run(wait.NeverStop)
	return eStore
}
