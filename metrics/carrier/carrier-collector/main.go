/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2018, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.ibm.com/alchemy-containers/armada-performance/metrics/carrier/config"
	"github.ibm.com/alchemy-containers/armada-performance/metrics/carrier/prometheus"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

const (
	// NamespaceMonitoring defines the monitoring namespace which is where the Prometheus pods reside
	namespaceMonitoring   string = "monitoring"
	prometheusServiceName string = "armada-ops-prometheus"
)

// initialize sets up and provides access to the carrier/tugboat Prometheus metrics
func initialize(stopChannel chan struct{}, readyChannel chan struct{}) {
	config.ParseConfig(filepath.Join(config.GetConfigPath(), "prom.toml"), &config.Prometheus)

	// Access Prometheus pod on the carrier via port forwarding
	cfg, err := clientcmd.BuildConfigFromFlags("", config.CarrierKubeconfig)
	if err != nil {
		log.Fatalf("Failed to build carrier kubeconfig: %s\n", err)
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to create clientset for carrier kubeconfig: %s\n", err)
	}

	// Get details of the Prometheus service
	svc, err := clientset.CoreV1().Services(namespaceMonitoring).Get(context.TODO(), prometheusServiceName, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Failed to find Prometheus service %s : %s\n", prometheusServiceName, err)
	}

	// Use the service selector to find the Prometheus pod(s)
	svcSelectorLabels := []string{}
	for k, v := range svc.Spec.Selector {
		svcSelectorLabels = append(svcSelectorLabels, strings.Join([]string{k, v}, "="))
	}
	promPodLabels := strings.Join(svcSelectorLabels, ",")
	pods, err := clientset.CoreV1().Pods(namespaceMonitoring).
		List(
			context.TODO(),
			metav1.ListOptions{LabelSelector: promPodLabels, Limit: 1},
		)
	if err != nil {
		log.Fatalf("Failed to find Prometheus pod(s): %s\n", err)

	}
	if len(pods.Items) == 0 {
		log.Fatalf("No pods found for service '%s'", prometheusServiceName)
	}

	// Check for a running pod
	prometheusPod := apiv1.Pod{}
	for _, pp := range pods.Items {
		if pp.Status.Phase == apiv1.PodRunning {
			prometheusPod = pp
			break
		}
	}
	if prometheusPod.GetName() == "" {
		log.Fatalln("Unable to forward port to Prometheus because no Prometheus pods are running. Current status=")
	}

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(namespaceMonitoring).
		Name(prometheusPod.GetName()).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(cfg)
	if err != nil {
		log.Fatalf("Failed to generate round tripper for port forward access : %s\n", err)
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())

	ow := io.Discard
	if config.Verbose {
		ow = os.Stdout
	}

	// Forward port from localhost to Promethues pod
	fw, err := portforward.New(
		dialer,
		[]string{fmt.Sprintf("%d:%d", config.Prometheus.Port, svc.Spec.Ports[0].TargetPort.IntValue())},
		stopChannel,
		readyChannel,
		ow, ow)
	if err != nil {
		log.Fatalf("Failed to create port forwarder: %s\n", err)
	}

	err = fw.ForwardPorts()
	if err != nil {
		log.Fatalf("Failed to forward ports: %s\n", err)
	}
}

func main() {
	var start, end time.Time
	var err error

	// Get start/end times and step interval for gathering metrics
	// Can generate time in correct format (on Linux) using 'date --iso-8601=seconds'
	startTime := flag.String("start", "", "Start time of period over which metrics are to be collected : RFC3339")
	endTime := flag.String("end", "", "End time of period over which metrics are to be collected : RFC3339")
	sampleInterval := flag.Duration("interval", 0, "Carrier Metrics collection step interval")
	reportingInterval := flag.Duration("frequency", 10*time.Minute, "Carrier Metrics collection step interval")

	flag.StringVar(&config.CarrierKubeconfig, "carrier", "", "Kubeconfig for carrier")

	flag.BoolVar(&config.Publish, "publish", false, "Publish carrier metrics to IBM Cloud monitoring service")
	flag.BoolVar(&config.Verbose, "verbose", false, "Verbose output required")

	flag.Parse()

	if len(config.CarrierKubeconfig) == 0 {
		log.Fatalln("Please specify location of Carrier Kubeconfig file")
	}

	if len(*startTime) == 0 {
		start = time.Now()
	} else {
		start, err = time.Parse(time.RFC3339, *startTime)
		if err != nil {
			log.Fatalf("Invalid start time '%s'. Specify in RFC3339 ('2006-01-02T15:04:05+07:00') format\n", *startTime)
		}
	}

	// Interval not specified by user, we'll come up with a sensible value
	if *sampleInterval == 0 {
		const (
			defaultSamples = 30
			minInterval    = 30 * time.Second
		)

		// If no interval specified let's default 30 values in total,
		// ensuring we don't collect more frequently than a minimum sample period
		// So for an hour long test, defaults would be one sample every 2 mins.
		*sampleInterval = *reportingInterval / defaultSamples
		if *sampleInterval < minInterval {
			*sampleInterval = minInterval
		}
	}

	if len(*endTime) == 0 {
		// Run forever. Well, one hundred years. (A sound like a tiger thrashing in the water)
		// Or in reality, the next failure/reboot
		end = time.Now().AddDate(100, 0, 0)
	} else {
		end, err = time.Parse(time.RFC3339, *endTime)
		if err != nil {
			log.Fatalf("Invalid end time '%s'. Specify in RFC3339 ('2006-01-02T15:04:05+07:00') format\n", *endTime)
		}
	}

	// Sanity check that end is after start
	if end.Before(start) {
		log.Fatalf("Invalid start/end times '%s' : '%s'. End time must be later than start time.\n", *startTime, *endTime)
	}

	log.Printf("Carrier/Tugboat Metrics\n- Start: %s\n- End: %s\n- Interval: %s\n\n", start.UTC(), end.UTC(), *sampleInterval)
	stopChannel := make(chan struct{}, 1)
	readyChannel := make(chan struct{})

	// Setup local access to carrier/tugboat Prometheus
	go initialize(stopChannel, readyChannel)
	<-readyChannel

	pc := prometheus.NewClient()
	for {
		// Wait for start time if necessary
		if tts := time.Until(start); tts > 0 {
			log.Printf("Metric collection start time %s not yet reached. Sleeping\n", start.Format(time.RFC3339))
			time.Sleep(tts)
		}

		stop := start
		for {
			next := stop.Add(*sampleInterval)

			if next.After(time.Now()) || next.After(end) {
				break
			}

			stop = next
		}

		log.Printf("Collecting metrics %s -> %s\n", start.Format("2006-01-02 15:04:05"), stop.Format("2006-01-02 15:04:05"))
		pc.GatherMetrics(start, stop, *sampleInterval)

		start = stop.Add(*sampleInterval)

		if start.After(end) {
			break
		}

		fmt.Println()
		time.Sleep(*reportingInterval)
	}
}
