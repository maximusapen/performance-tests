/*
Copyright 2016,2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
 launch.go

 Launch the netperf tests

 1. Launch the netperf-orch service
 2. Launch the worker pods
 3. Wait for the output csv data to show up in orchestrator pod logs
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	debugLog         = "output.txt"
	testNamespace    = "k8s-netperf"
	csvDataMarker    = "GENERATING CSV OUTPUT"
	csvEndDataMarker = "END CSV DATA"
	runUUID          = "latest"
	orchestratorPort = 5202
	iperf3Port       = 5201
	netperfPort      = 12865

	zoneLabel = "ibm-cloud.kubernetes.io/zone"

	//The maximum time to wait for results before giving up
	maxWaitTime = time.Minute * 60
)

var (
	iterations     int
	hostnetworking bool
	multizone      bool
	cleanupAtEnd   bool
	tag            string
	kubeConfig     string
	netperfImage   string

	everythingSelector = metav1.ListOptions{}

	primaryNode   apiv1.Node
	secondaryNode apiv1.Node
)

func init() {
	flag.BoolVar(&hostnetworking, "hostnetworking", false,
		"(boolean) Enable Host Networking Mode for PODs")
	flag.BoolVar(&cleanupAtEnd, "cleanup", true,
		"(boolean) Cleanup Kube resources after test completion")

	flag.IntVar(&iterations, "iterations", 1,
		"Number of iterations to run")
	flag.StringVar(&tag, "tag", runUUID, "CSV file suffix")
	flag.StringVar(&netperfImage, "image", "stg.icr.io/armada_performance/k8s-netperf", "Docker image used to run the network tests")
	flag.StringVar(&kubeConfig, "kubeConfig", "",
		"Location of the kube configuration file ($HOME/.kube/config")
}

func setupClient() *kubernetes.Clientset {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	return clientset
}

// getMinions : Only return schedulable/worker nodes
func getMinionNodes(c *kubernetes.Clientset) *apiv1.NodeList {
	nodes, err := c.CoreV1().Nodes().List(
		context.TODO(),
		metav1.ListOptions{
			FieldSelector: "spec.unschedulable=false",
		})
	if err != nil {
		fmt.Println("Failed to fetch nodes", err)
		return nil
	}
	return nodes
}

func cleanup(c *kubernetes.Clientset) {
	// Cleanup existing rcs, pods and services in our namespace
	rcs, err := c.CoreV1().ReplicationControllers(testNamespace).List(context.TODO(), everythingSelector)
	if err != nil {
		fmt.Println("Failed to get replication controllers", err)
		return
	}
	for _, rc := range rcs.Items {
		fmt.Println("Deleting rc", rc.GetName())
		if err := c.CoreV1().ReplicationControllers(testNamespace).Delete(
			context.TODO(),
			rc.GetName(), metav1.DeleteOptions{}); err != nil {
			fmt.Println("Failed to delete rc", rc.GetName(), err)
		}
	}
	pods, err := c.CoreV1().Pods(testNamespace).List(context.TODO(), everythingSelector)
	if err != nil {
		fmt.Println("Failed to get pods", err)
		return
	}
	for _, pod := range pods.Items {
		fmt.Println("Deleting pod", pod.GetName())
		if err := c.CoreV1().Pods(testNamespace).Delete(context.TODO(), pod.GetName(), metav1.DeleteOptions{GracePeriodSeconds: new(int64)}); err != nil {
			fmt.Println("Failed to delete pod", pod.GetName(), err)
		}
	}
	svcs, err := c.CoreV1().Services(testNamespace).List(context.TODO(), everythingSelector)
	if err != nil {
		fmt.Println("Failed to get services", err)
		return
	}
	for _, svc := range svcs.Items {
		fmt.Println("Deleting svc", svc.GetName())
		c.CoreV1().Services(testNamespace).Delete(
			context.TODO(),
			svc.GetName(), metav1.DeleteOptions{})
	}
}

// createServices: Long-winded function to programmatically create our two services
func createServices(c *kubernetes.Clientset) bool {
	// Create our namespace if not present
	if _, err := c.CoreV1().Namespaces().Get(context.TODO(), testNamespace, metav1.GetOptions{}); err != nil {
		c.CoreV1().Namespaces().Create(context.TODO(), &apiv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}, metav1.CreateOptions{})
	}

	// Create the orchestrator service that points to the coordinator pod
	orchLabels := map[string]string{"app": "netperf-orch"}
	orchService := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "netperf-orch",
		},
		Spec: apiv1.ServiceSpec{
			Selector: orchLabels,
			Ports: []apiv1.ServicePort{{
				Name:       "netperf-orch",
				Protocol:   apiv1.ProtocolTCP,
				Port:       orchestratorPort,
				TargetPort: intstr.FromInt(orchestratorPort),
			}},
			Type: apiv1.ServiceTypeClusterIP,
		},
	}
	if _, err := c.CoreV1().Services(testNamespace).Create(context.TODO(), orchService, metav1.CreateOptions{}); err != nil {
		fmt.Println("Failed to create orchestrator service", err)
		return false
	}
	fmt.Println("Created orchestrator service")

	// Create the netperf-w2 service that points a clusterIP at the worker 2 pod
	netperfW2Labels := map[string]string{"app": "netperf-w2"}
	netperfW2Service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "netperf-w2",
		},
		Spec: apiv1.ServiceSpec{
			Selector: netperfW2Labels,
			Ports: []apiv1.ServicePort{
				{
					Name:       "netperf-w2",
					Protocol:   apiv1.ProtocolTCP,
					Port:       iperf3Port,
					TargetPort: intstr.FromInt(iperf3Port),
				},
				{
					Name:       "netperf-w2-udp",
					Protocol:   apiv1.ProtocolUDP,
					Port:       iperf3Port,
					TargetPort: intstr.FromInt(iperf3Port),
				},
				{
					Name:       "netperf-w2-netperf",
					Protocol:   apiv1.ProtocolTCP,
					Port:       netperfPort,
					TargetPort: intstr.FromInt(netperfPort),
				},
			},
			Type: apiv1.ServiceTypeClusterIP,
		},
	}
	if _, err := c.CoreV1().Services(testNamespace).Create(context.TODO(), netperfW2Service, metav1.CreateOptions{}); err != nil {
		fmt.Println("Failed to create netperf-w2 service", err)
		return false
	}
	fmt.Println("Created netperf-w2 service")
	return true
}

// createRCs - Create replication controllers for all workers and the orchestrator
func createRCs(c *kubernetes.Clientset) bool {
	// Create the orchestrator RC
	name := "netperf-orch"
	fmt.Println("Creating replication controller", name)
	replicas := int32(1)

	_, err := c.CoreV1().ReplicationControllers(testNamespace).Create(context.TODO(), &apiv1.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: apiv1.ReplicationControllerSpec{
			Replicas: &replicas,
			Selector: map[string]string{"app": name},
			Template: &apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: apiv1.PodSpec{
					HostNetwork: hostnetworking,
					Containers: []apiv1.Container{
						{
							Name:            name,
							Image:           netperfImage,
							Ports:           []apiv1.ContainerPort{{ContainerPort: orchestratorPort}},
							Args:            []string{"--mode=orchestrator"},
							ImagePullPolicy: "Always",
						},
					},
					TerminationGracePeriodSeconds: new(int64),
				},
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		fmt.Println("Error creating orchestrator replication controller", err)
		return false
	}
	fmt.Println("Created orchestrator replication controller")
	for i := 1; i <= 3; i++ {
		// Bring up pods slowly
		time.Sleep(3 * time.Second)
		kubeNode := primaryNode.GetName()
		if i == 3 {
			kubeNode = secondaryNode.GetName()
		}
		name = fmt.Sprintf("netperf-w%d", i)
		fmt.Println("Creating replication controller", name)
		portSpec := []apiv1.ContainerPort{}
		if i > 1 {
			// Worker W1 is a client-only pod - no ports are exposed
			portSpec = append(portSpec, apiv1.ContainerPort{ContainerPort: iperf3Port, Protocol: apiv1.ProtocolTCP})
		}

		workerEnv := []apiv1.EnvVar{
			{Name: "worker", Value: name},
			{Name: "kubeNode", Value: kubeNode},
			{Name: "podname", Value: name},
		}

		replicas := int32(1)

		_, err := c.CoreV1().ReplicationControllers(testNamespace).Create(context.TODO(), &apiv1.ReplicationController{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: apiv1.ReplicationControllerSpec{
				Replicas: &replicas,
				Selector: map[string]string{"app": name},
				Template: &apiv1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": name},
					},
					Spec: apiv1.PodSpec{
						NodeName:    kubeNode,
						HostNetwork: hostnetworking,
						Containers: []apiv1.Container{
							{
								Name:            name,
								Image:           netperfImage,
								Ports:           portSpec,
								Args:            []string{"--mode=worker"},
								Env:             workerEnv,
								ImagePullPolicy: "Always",
							},
						},
						TerminationGracePeriodSeconds: new(int64),
					},
				},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			fmt.Println("Error creating orchestrator replication controller", name, ":", err)
			return false
		}
	}

	return true
}

func getOrchestratorPodName(pods *apiv1.PodList) string {
	for _, pod := range pods.Items {
		if strings.Contains(pod.GetName(), "netperf-orch-") {
			return pod.GetName()
		}
	}
	return ""
}

func getPodName(pods *apiv1.PodList, podName string) string {
	for _, pod := range pods.Items {
		if strings.Contains(pod.GetName(), podName) {
			return pod.GetName()
		}
	}
	return ""
}

// Retrieve the logs for the pod/container and check if csv data has been generated
func getCsvResultsFromPod(c *kubernetes.Clientset, podName string) *string {
	body, err := c.CoreV1().Pods(testNamespace).GetLogs(podName, &apiv1.PodLogOptions{Timestamps: false}).DoRaw(context.TODO())
	if err != nil {
		fmt.Printf("Error (%s) reading logs from pod %s", err, podName)
		return nil
	}
	logData := string(body)
	index := strings.Index(logData, csvDataMarker)
	endIndex := strings.Index(logData, csvEndDataMarker)
	if index == -1 || endIndex == -1 {
		return nil
	}
	csvData := string(body[index+len(csvDataMarker)+1 : endIndex])
	return &csvData
}

// Retrieve the logs for the pod/container and check if csv data has been generated
func printPodLogs(c *kubernetes.Clientset, labelName string) {
	if pods, err := c.CoreV1().Pods(testNamespace).List(context.TODO(), everythingSelector); err == nil {
		podName := getPodName(pods, labelName)
		body, err := c.CoreV1().Pods(testNamespace).GetLogs(podName, &apiv1.PodLogOptions{}).DoRaw(context.TODO())
		if err != nil {
			fmt.Printf("Error (%s) reading logs from pod %s", err, podName)
			return
		}
		logData := string(body)
		fmt.Printf("Printing logs from pod %s :\n %s", podName, logData)

	} else {
		fmt.Printf("Error getting logs from pod matching %s: %s", labelName, err)
	}
}

// processCsvData : Process the CSV datafile and generate line and bar graphs
func processCsvData(csvData *string) bool {
	outputFilePrefix := fmt.Sprintf("%s-%s.", testNamespace, tag)
	fmt.Printf("Test concluded - CSV raw data written to %s.csv\n", outputFilePrefix)
	fd, err := os.OpenFile(fmt.Sprintf("%scsv", outputFilePrefix), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		fmt.Println("ERROR writing output CSV datafile", err)
		return false
	}
	fd.WriteString(fmt.Sprintf("host networking,%t\n", hostnetworking))
	fd.WriteString(fmt.Sprintf("multizone,%t\n", multizone))
	fd.WriteString(*csvData)
	fd.Close()
	return true
}

func executeTests(c *kubernetes.Clientset) bool {
	for i := 0; i < iterations; i++ {
		cleanup(c)
		if !createServices(c) {
			fmt.Println("Failed to create services - aborting test")
			return false
		}
		time.Sleep(3 * time.Second)
		if !createRCs(c) {
			fmt.Println("Failed to create replication controllers - aborting test")
			return false
		}
		fmt.Println("Waiting for netperf pods to start up")

		var orchestratorPodName string
		waitStartTime := time.Now()
		for len(orchestratorPodName) == 0 {
			fmt.Println("Waiting for orchestrator pod creation")
			time.Sleep(60 * time.Second)
			var pods *apiv1.PodList
			var err error
			if time.Since(waitStartTime) > maxWaitTime {
				fmt.Printf("Timed out waiting for orchestrator pod creation after %v ", maxWaitTime)
				return false
			}
			if pods, err = c.CoreV1().Pods(testNamespace).List(context.TODO(), everythingSelector); err != nil {
				fmt.Println("Failed to fetch pods - waiting for pod creation", err)
				continue
			}
			orchestratorPodName = getOrchestratorPodName(pods)
		}
		fmt.Println("Orchestrator Pod is", orchestratorPodName)

		// The pods orchestrate themselves, we just wait for the results file to show up in the orchestrator container
		waitStartTime = time.Now()
		for true {
			// Monitor the orchestrator pod for the CSV results file
			csvdata := getCsvResultsFromPod(c, orchestratorPodName)
			if csvdata == nil {
				fmt.Println("Scanned orchestrator pod filesystem - no results file found yet...waiting for orchestrator to write CSV file...")
				if time.Since(waitStartTime) > maxWaitTime {
					fmt.Printf("Timed out waiting for results after %v ", maxWaitTime)
					return false
				}
				time.Sleep(60 * time.Second)
				continue
			}
			if processCsvData(csvdata) {
				break
			}
		}
		fmt.Printf("TEST RUN (Iteration %d) FINISHED - cleaning up services and pods\n", i)
	}
	return true
}

func main() {
	flag.Parse()
	fmt.Println("Network Performance Test")
	fmt.Println("Parameters :")
	fmt.Println("Iterations      : ", iterations)
	fmt.Println("Host Networking : ", hostnetworking)
	fmt.Println("Docker image    : ", netperfImage)
	fmt.Println("------------------------------------------------------------")

	var c *kubernetes.Clientset
	if c = setupClient(); c == nil {
		fmt.Println("Failed to setup REST client to Kubernetes cluster")
		return
	}

	nodes := getMinionNodes(c)
	if nodes == nil || len(nodes.Items) < 2 {
		fmt.Println("Insufficient number of nodes for test (need minimum 2 nodes)")
		fmt.Println("Netperf Test Failed")
		os.Exit(1)
	}

	zones := make(map[string]apiv1.Node)
	for _, n := range nodes.Items {
		zone := n.GetLabels()[zoneLabel]
		zones[zone] = n
	}

	if len(zones) > 1 {
		// Nodes are in a multiple zones, ensure primary and secondary are in separate zones
		multizone = true
		for _, n := range zones {
			if primaryNode.GetName() == "" {
				primaryNode = n
			} else {
				secondaryNode = n
				break
			}
		}
	} else {
		// Nodes are in a single zone
		primaryNode = nodes.Items[0]
		secondaryNode = nodes.Items[1]
	}

	fmt.Printf("Selected primary node = (%s, %s)\n", primaryNode.GetName(), primaryNode.GetLabels()[zoneLabel])
	fmt.Printf("Selected secondary node = (%s, %s)\n", secondaryNode.GetName(), secondaryNode.GetLabels()[zoneLabel])

	successfulTest := executeTests(c)

	// Dump the logs from the netperf-w1 pod as that contains the iperf output
	printPodLogs(c, "netperf-w1-")

	if cleanupAtEnd {
		cleanup(c)
	}

	if !successfulTest {
		fmt.Println("Netperf Test Failed")
		os.Exit(1)
	}
}
