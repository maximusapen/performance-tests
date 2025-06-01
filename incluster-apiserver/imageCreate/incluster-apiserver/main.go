/*******************************************************************************
 *
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2020, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets, irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

/*
Appication to drive load against an apiserver from within the cluster
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"encoding/json"

	"golang.org/x/time/rate"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Metrics - Structure to hold the metrics
type Metrics struct {
	Name              string
	Timestamp         string
	RunTime           int
	NumRequests       int
	NumSuccess        int
	NumErrors         int
	NumSoftErrors     int // A SoftError is when we saw an error, then retried and the retry succeeds (Can be caused by apiserver restarting)
	Throughput        float64
	MinResponseTime   int64
	MeanResponseTime  int64
	MaxResponseTime   int64
	TotalResponseTime int64
	ItemsPerResponse  int
}

var (
	throughput         int
	namespace          string
	runtime            int
	disableCompression bool
)

func main() {

	flag.IntVar(&throughput, "throughput", 0, "The throughput in requests per second to limit requests")
	flag.StringVar(&namespace, "namespace", "incluster-apiserver-target", "The namespace in which to make requests")
	flag.IntVar(&runtime, "runtime", 300, "The duration of the test in seconds")
	flag.BoolVar(&disableCompression, "disable_compression", false, "Whether to disable compression or large response from the apiserver")

	flag.Parse()

	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	// Defaults are QPS:5 Burst:10 - so need to override these
	config.QPS = 10000
	config.Burst = 1000
	config.DisableCompression = disableCompression

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	var rateLimit rate.Limit
	if throughput == 0 {
		fmt.Print("Running with unlimited throughput \n")
		rateLimit = rate.Inf
	} else {
		fmt.Printf("Running with throughput limited at %v requests per second\n", throughput)
		rateLimit = rate.Limit(float64(throughput))
	}
	fmt.Printf("Using namespace %s to make requests, running for %d seconds, disable_compression is %t \n", namespace, runtime, disableCompression)

	// Setup rate limiter to control throughput
	limiter := rate.NewLimiter(rateLimit, 1)

	startTime := time.Now()

	var numItems int = 0
	summaryMetrics := &Metrics{Name: "Summary"}
	for {
		limiter.Wait(context.Background())
		summaryMetrics.NumRequests++

		var start time.Time
		var respTime int64
		var err error
		var pods *v1.PodList

		// Just do Pods
		// Used to do Get Secrets too - but Openshift has several large secrets in each namespace, so we see much worse
		// performance for Openshift
		start = time.Now()
		pods, err = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
		respTime = time.Since(start).Milliseconds()
		numItems += len(pods.Items)

		if err != nil {
			// If an apiserver restarts it can cause a single failure that will succeed on retry, so we do not want to count these as
			// full "Errors" - so count as a "SoftError" if the retry succeeds
			err = nil
			start = time.Now()
			pods, err = clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
			respTime = time.Since(start).Milliseconds()
			numItems += len(pods.Items)
			if err != nil {
				fmt.Printf("Error occurred: %v \n", err)
				summaryMetrics.NumErrors++
			} else {
				fmt.Printf("Error occurred, but retry succeeded, counting as SoftError: %v \n", err)
				summaryMetrics.NumSoftErrors++
			}
		} else {
			summaryMetrics.NumSuccess++
		}

		summaryMetrics.TotalResponseTime += respTime
		if summaryMetrics.MinResponseTime == 0 || respTime < summaryMetrics.MinResponseTime {
			summaryMetrics.MinResponseTime = respTime
		}
		if respTime > summaryMetrics.MaxResponseTime {
			summaryMetrics.MaxResponseTime = respTime
		}
		// Print metrics every 100 requests
		if summaryMetrics.NumRequests%100 == 0 {
			currentTime := time.Now()
			timeSoFar := currentTime.Sub(startTime)
			summaryMetrics.RunTime = int(timeSoFar.Seconds())
			summaryMetrics.Timestamp = currentTime.Format("02-01-2006 15:04:05")

			summaryMetrics.Throughput = float64(summaryMetrics.NumRequests) / timeSoFar.Seconds()
			summaryMetrics.MeanResponseTime = summaryMetrics.TotalResponseTime / int64(summaryMetrics.NumRequests)
			summaryMetrics.ItemsPerResponse = numItems / summaryMetrics.NumRequests

			if pods != nil {
				fmt.Printf("There are %d pods in the %s namespace\n", len(pods.Items), namespace)
			}

			// Print the result as json
			b, err := json.Marshal(summaryMetrics)
			if err != nil {
				fmt.Printf("Error occurred marshalling results to json : %v", err)
				return
			}
			fmt.Println(string(b))

			// Test has finished so break out of the loop
			if summaryMetrics.RunTime > runtime {
				break
			}
		}
	}
	// Wait so the results can be collected
	fmt.Println("Test has finished, waiting for results to be collected")
	time.Sleep(time.Duration(10) * time.Minute)
}
