/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2018, 2020 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	metricsservice "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.ibm.com/alchemy-containers/armada-performance/metrics/cruiser/controller"
	"github.ibm.com/alchemy-containers/armada-performance/metrics/cruiser/endpoints"
)

var verbose, publish bool

func collect(resources []endpoints.Resource) {
	for _, r := range resources {
		res := r.Metrics()

		// Generate metrics for use with IBM Cloud monitoring service
		if res != nil {
			if verbose {
				fmt.Println(res)
			}

			bm := res.BMMetrics()

			// Send to metrics serivce if requested - razee alerts not required
			if publish {
				metricsservice.WriteBluemixMetrics(bm, true, "", "")
			} else {
				fmt.Println(bm)
				fmt.Println()
			}
		}
	}
}

func main() {
	// Include filename and line number in fatal error messages
	fatalLog := log.New(os.Stderr, log.Prefix(), log.LstdFlags|log.Lshortfile)

	kubeconfig := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	delay := flag.Duration("delay", 0, "metrics collection delay period")
	interval := flag.Duration("interval", 60*time.Second, "metrics collection step interval")
	controlPort := flag.Int("controlPort", 20569, "local controller control port")

	flag.BoolVar(&verbose, "verbose", false, "Verbose output required")
	flag.BoolVar(&publish, "publish", false, "Publish carrier metrics to IBM Cloud monitoring service")

	flag.StringVar(&endpoints.Filter, "filter", "", "regular expression to limit metrics to matching namespace/pod/container names")
	flag.StringVar(&endpoints.Level, "level", "namespace", "level at which to collect metrics (node, namespace, pod or container)")
	flag.StringVar(&endpoints.Testname, "test", "", "Test name to be included in grafana metrics name")

	flag.Parse()

	// Build Kubernetes config from kubeconfig filepath....
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		fatalLog.Fatalln(err.Error())
	}

	// ....and then create Kubernetes clientset from this config
	endpoints.KubeClientset, err = kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		fatalLog.Fatalln(err.Error())
	}

	var resources []endpoints.Resource

	// Levels hierarchy is: Node -> Namespace -> Pod -> Container
	// We'll always collect metrics at the node level, and then calculate aggregated values at the requested level
	switch endpoints.Level {
	case "namespace", "pod", "container":
		resources = append(resources, &endpoints.PodMetrics{})
		fallthrough
	case "node":
		resources = append(resources, &endpoints.NodeMetrics{})
	default:
		fatalLog.Fatalf("Unrecognized level '%s'\n", endpoints.Level)
	}

	// Create communication channel and start the controller
	controlChan := make(chan controller.CommandType)
	go controller.Start(*controlPort, controlChan)

	var afterChan, tickerChan <-chan time.Time
	var ticker *time.Ticker

	for terminate := false; !terminate; {
		select {
		case <-afterChan:
			// Initial '--delay' period complete. Fire off future collections every '--interval'
			ticker = time.NewTicker(*interval)
			tickerChan = ticker.C

			// and collect initial set of metrics
			collect(resources)

		case <-tickerChan:
			// Interval ticker fired, collect metrics
			collect(resources)

		case control := <-controlChan:
			switch control {
			case controller.START:
				// Request to start metric collection received. Start collection after specified delay period.
				afterChan = time.After(*delay)

			case controller.STOP:
				// Request to stop metric collection recieved. Stop ticker but continue to listen for further commands.
				if ticker != nil {
					ticker.Stop()
				}

			case controller.TERMINATE:
				// Terminate request received. Metrics collection tool will exit immediately.
				terminate = true

			default:
				// Shrug - should never receive invalid commands from controller.
				fatalLog.Fatalf("Unrecognized control byte %x\n", byte(control))
			}
		}
	}
}
