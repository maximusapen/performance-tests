/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2021 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/cluster"
	"github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/config"
	"github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/request"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
)

var (
	clientset     *kubernetes.Clientset
	clusterConfig cluster.Cluster
	orgID         string
	spaceID       string
	clusterName   string
	clusterPrefix string
	totalWorkers  int
	terminate     bool
	signalCount   int
	backgroundApp string
	activeApp     string
	accessPoints  []string
	conf          *config.Config
	configPath    string
	testsToRun    []bool
	// Debug enables verbose messages
	Debug bool
)

// Index into arrays defining which tests should be run
const (
	TestArmadaAccessible  int = 0
	TestArmadaFunctional  int = 1
	TestCruiserAccessible int = 2
	TestCruiserFunctional int = 3
	TestAppAccessible     int = 4
)

// NumberOfTests returns the total number of supported tests
// To be used to properly size inTestsToRun
func NumberOfTests() int {
	return 5
}

// Run executes the specified tests
func Run(inConf *config.Config, inClusterPrefix string, inTotalWorkers int,
	inActiveApp string, inBackgroundApp string, inTestsToRun []bool) {
	fmt.Println("Configuring test environment")

	signalCount = 0

	clusterPrefix = inClusterPrefix
	totalWorkers = inTotalWorkers
	backgroundApp = inBackgroundApp
	activeApp = inActiveApp
	conf = inConf
	testsToRun = inTestsToRun

	configPath = config.GetConfigPath()

	cluster.Debug = Debug

	// Generate a unique cluster name
	clusterName = fmt.Sprintf("%s%d", clusterPrefix, 1)

	// Check if cluster exists, if not then create
	cls, err := cluster.GetClusters()
	if err != nil {
		panic(err)
	}

	for _, c := range cls {
		if strings.Compare(clusterName, c.Name) == 0 {
			fmt.Printf("Found master cluster: %s\n", c.Name)
			clusterConfig = c
		} else if testsToRun[TestArmadaFunctional] && strings.HasPrefix(c.Name, clusterPrefix) {
			fmt.Printf("Found non master prefix. Delete: %s\n", c.Name)
			c.Delete()
		}
	}

	if clusterConfig.Name == "" {
		clusterConfig, err = cluster.CreateCluster(clusterName, totalWorkers)
		if err != nil {
			panic(err)
		}
	}

	fmt.Println("Wait for cluster to be deployed")
	if clusterConfig.WaitForDeployed() == false {
		panic("Master cluster didn't reach 'deployed' state within prescribed time")
	}
	if clusterConfig.WaitForWorkers(60) == false {
		panic("Cluster workers didn't reach 'deployed' state within prescribed time")
	}
	clientset = clusterConfig.GetKubeClientSet()

	// TODO if just created cluster then create client app
	// App must have accessible NodePorts to ping and have a server running on each node.
	if testsToRun[TestAppAccessible] && appServiceAccessible() != "success" {
		panic("Client app isn't accessible. This needs to be fix before running tests.")
	}

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT)

	go func() {
		<-sigs
		terminate = true
		signalCount++
		if signalCount > 2 {
			panic("Exiting due to multiple SIGTIN/^c")
		}
		fmt.Println("Program will terminate at next interupt point")
		signal.Notify(sigs, syscall.SIGINT)
	}()

	fmt.Printf("Starting test: %v\n", time.Now().Format("2006-01-02 15:04:05"))

	fmt.Printf("Time, Delta Time, %v, %v, %v, %v, %v\n", "apiAcc", "apiFun", "cruAcc", "cruFun", "appAcc")
	var apiAcc = "n/a"
	var apiFun = "n/a"
	var cruAcc = "n/a"
	var cruFun = "n/a"
	var appAcc = "n/a"
	for {
		var start = time.Now()
		if testsToRun[TestArmadaAccessible] {
			apiAcc = apiServiceAccessible()
			if terminate {
				break
			}
		}
		if testsToRun[TestArmadaFunctional] {
			apiFun = apiServiceFunctional()
			if terminate {
				break
			}
		}
		if testsToRun[TestCruiserAccessible] {
			cruAcc = cruiserServiceAccessible()
			if terminate {
				break
			}
		}
		if testsToRun[TestCruiserFunctional] {
			cruFun = cruiserServiceFunctional()
			if terminate {
				break
			}
		}
		if testsToRun[TestAppAccessible] {
			appAcc = appServiceAccessible()
			if terminate {
				break
			}
		}
		fmt.Printf("%v, %v, %v, %v, %v, %v, %v\n", time.Now().Format("2006-01-02 15:04:05"), time.Since(start), apiAcc, apiFun, cruAcc, cruFun, appAcc)

		if testsToRun[1] == false {
			time.Sleep(20 * time.Second)
		}
	}
}

// Simple request to make sure api server is active
func apiServiceAccessible() string {
	if Debug {
		fmt.Println("apiServiceAccessible()")
	}
	var result = "failed"

	api := request.Data{Action: config.ActionGetDatacenters}
	r := request.PerformRequest(api, true)

	if r.StatusCode == http.StatusOK {
		var dat []string
		if err := json.Unmarshal(r.Body, &dat); err != nil {
			panic(err)
		}
		result = "success"
	}
	return result
}

// Create a cluster and use kube to make sure it exists, then destroy it
func apiServiceFunctional() string {
	if Debug {
		fmt.Println("apiServiceFunctional()")
	}
	var result = "failed"

	cl, err := cluster.CreateCluster(clusterPrefix+"2", 1)
	if err != nil {
		fmt.Println(err)
		result = "failed: err on create"
	} else {
		if cl.WaitForDeployed() {
			cl.GetKubeClientSet()
			if cl.Created() {
				result = "success"
			} else {
				result = "failed: not created"
			}
			cl.Delete()
			cl.WaitForDeleted()
		}
	}
	return result
}

// Use kubeClientset to check that cluster exists
func cruiserServiceAccessible() string {
	if Debug {
		fmt.Println("cruiserServiceAccessible()")
	}
	if clusterConfig.Created() {
		return "success"
	}
	return "failed"
}

// Create a client app
func cruiserServiceFunctional() string {
	if Debug {
		fmt.Println("cruiserServiceFunctional()")
	}

	var result = "failed"

	parts, err := clusterConfig.CreateApp(filepath.Join(configPath, activeApp+".yml"))

	if err == nil {

	podloop:
		for {
			ok, statuses := clusterConfig.GetAppStatus(parts)
			if ok {
				if Debug {
					fmt.Println("appServiceAccessible() App is deployed")
				}
				result = "success"
				break
			}
			for part, status := range statuses {
				switch status {
				case "Failed":
					if Debug {
						fmt.Println(status)
					}
					break podloop
					/*
						case "Running":
						case "Succeeded":
							break podloop
						case "Unknown":
							fmt.Println(status)
						case "Pending":
					*/
				default:
					if Debug {
						fmt.Println("App part status: ", part, status)
					}
				}
			}
			time.Sleep(5 * time.Second)
		}
	}

	clusterConfig.DeleteApp(parts)
	clusterConfig.WaitDeletedApp(parts)
	return result
}

// Ping client app
func appServiceAccessible() string {
	if Debug {
		fmt.Println("appServiceAccessible()")
	}
	var result = "failed"

	getAppDetails()

	if len(accessPoints) > 0 {

		// Dynamically find ip & port from NodePort setting
		conn, err := net.Dial("tcp", accessPoints[0])
		if err != nil {
			fmt.Println("Connection error:", err)
			result = "failed: connect err"
		} else {
			conn.Close()
			result = "success"
		}

	}
	return result
}

// Get app details from api first time through
// Assumption these are NodePorts with TCP connection
func getAppDetails() []string {
	//var accessPoints = [...]string{"169.55.8.34:31593"}

	if len(accessPoints) == 0 {
		var nodePorts []int32

		serviceName := backgroundApp + "-public"
		fmt.Println("Public service name", serviceName)

		// Get port info
		// TODO add service filter based on name
		services, err := clientset.CoreV1().Services(k8sv1.NamespaceDefault).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Println("ERROR: Services name ", serviceName, err)
		} else if len(services.Items) == 0 {
			fmt.Println("No services found for ", serviceName)
		} else {
			for _, svc := range services.Items {
				if svc.Name == serviceName {
					for _, svcPort := range svc.Spec.Ports {
						if Debug {
							fmt.Println("NodePort ", svcPort.NodePort)
						}
						if svcPort.NodePort > 0 {
							nodePorts = append(nodePorts, svcPort.NodePort)
						}
					}
				}
			}
		}

		// Get list of worker IPs
		nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			fmt.Println("ERROR: Nodes ", clusterName, err)
		} else if len(nodes.Items) == 0 {
			fmt.Println("No nodes found for ", clusterName)
		} else {
			for _, node := range nodes.Items {
				for _, nodeAddress := range node.Status.Addresses {
					if nodeAddress.Type == k8sv1.NodeExternalIP {
						if Debug {
							fmt.Println("Node IP ", nodeAddress.Address)
						}
						for _, port := range nodePorts {
							accessPoints = append(accessPoints, nodeAddress.Address+":"+strconv.FormatInt(int64(port), 10))
						}
					}
				}
			}
		}
	}

	return accessPoints
}
