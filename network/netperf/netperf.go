/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2017, 2022 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

//
// Process
// - Define KUBECONFIG to point to the carrier
// - Setup .ssh/config so all baremetal and VSI hosts can be logged into directly with "ssh <host IP or name>"
//   ->  Be aware that those hosts probably don't have DNS lookup so using IPs makes sense
// - Deploy containers if part of the test
//   kubectl create -f <yaml file>
// - Install iperf3 on internet, vsi and baremetal hosts. It is done automatically for containers.
//   sudo apt-get update
//   sudo apt-get install iperf3
// - example:
//   ./netperf -hosts "vsi1:stage-dal09-carrier1-worker-03,vsi2:10.143.115.225" -tests "iperf3:vsi1:vsi2" -output results.json
//   iperf3 Result: Wed, 09 Aug 2017 14:11:41 GMT, vsi1 -> vsi2 sent 6.4 GB, avg rate 5.1 Gbits/sec
// - example of merging files:
//   ./netperf -output merge.json -merge tmp.json,tmp2.json
//   iperf3 Result: Wed, 09 Aug 2017 14:42:07 GMT, vsi1 -> vsi2 sent 6.1 GB, avg rate 4.8 Gbits/sec

// TODO
// Handle iperf3 sum_sent.retransmits
// Flush out iperf3 objects so full set of data is stored
// Handle pod -> ClusterIP by logging into nodeport or loadbalancer and running test to ClusterIP
// Deploy services in k8s if don't exit
// Reduce exposure of port 22 in container. If could avoid login to pod for ClusterIP test then that could solve problem.
// Deploy iperf3 in ubuntu hosts as needed (Probably not)
//     sudo apt-get update
//     sudo apt-get install iperf3

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	clientcmd "k8s.io/client-go/tools/clientcmd"
)

// Pod represents a kube pod
type Pod struct {
	Name   string
	NodeIP string
}

// Endpoint for tests
type Endpoint struct {
	Name string
	Type EndpointType
	IP   string
	Port int32
	Pods []Pod
}

// EndpointType ...
type EndpointType string

const (
	// EndpointTypeClusterIP is for kube ClusterIP service
	EndpointTypeClusterIP EndpointType = "ClusterIP"
	// EndpointTypeNodePort is for kube NodePort service
	EndpointTypeNodePort EndpointType = "NodePort"
	// EndpointTypeLoadBalancer is for kube LoadBalancer service
	EndpointTypeLoadBalancer EndpointType = "LoadBalancer"
	// EndpointTypeHost is Linux hosts
	EndpointTypeHost EndpointType = "Host"
)

// TestType defines the test supported
type TestType string

// The types of tests that are supported
const (
	Iperf3Test TestType = "iperf3"
	PingTest   TestType = "ping"
)

// Results stores the context for the test, the test definitions and results
type Results struct {
	Endpoints map[string]Endpoint
	Tests     []Test
}

// Test definition and results
type Test struct {
	Client   string
	Server   string
	TestType TestType
	Time     string
	Error    string
	Iperf3   Iperf3
	Ping     Ping
}

// The following types are used while parsing json output of iperf

// Iperf3 represents a single run of iperf3
type Iperf3 struct {
	Start     Start
	Intervals interface{}
	End       End
	Error     string
}

// Start the conditions for iperf3 test
type Start struct {
	Connected     interface{}
	ConnectingTo  interface{} `json:"connection_to"`
	Cookie        interface{}
	SystemInfo    string      `json:"system_info"`
	TCPMssDefault uint        `json:"tcp_mss_default"`
	TestStart     interface{} `json:"test_start"`
	Timestamp     Timestamp
	Version       string
}

// Timestamp the start time for iperf3 test
type Timestamp struct {
	Time     string
	Timesecs float32
}

// End detailed and summary results of an iperf3 test
type End struct {
	Streams               []Stream
	SumSent               Summary        `json:"sum_sent"`
	SumReceived           Summary        `json:"sum_received"`
	CPUUtilizationPercent CPUUtilization `json:"cpu_utilization_percent"`
}

// Stream from iperf3 test
type Stream struct {
	Sender   Result
	Receiver Result
}

// Summary of iperf3 test
type Summary struct {
	Start         uint64
	End           float64
	Seconds       float64
	Bytes         uint64
	BitsPerSecond float64 `json:"bits_per_second"`
	Retransmits   uint64
}

// CPUUtilization from iperf3 test
type CPUUtilization struct {
	HostTotal    float64 `json:"host_total"`
	HostUser     float64 `json:"host_user"`
	HostSystem   float64 `json:"host_system"`
	RemoteTotal  float64 `json:"remote_total"`
	RemoteUser   float64 `json:"remote_user"`
	RemoteSystem float64 `json:"remote_system"`
}

// Result of iperf3 test
type Result struct {
	Socket        uint64
	Start         uint64
	End           float64
	Seconds       float64
	Bytes         uint64
	BitsPerSecond float64 `json:"connection_to"`
}

// Indices for dynamic variables in iperf3Args
const (
	HostArg = 0
	IPArg   = 3
	PortArg = 5
)

// Ping ...
type Ping struct {
	InterPacketGap   float32
	EwmAvg           float32 // Exponential Weight Moving Average
	Received         uint32
	RttMin           float32
	RttAvg           float32
	RttMax           float32
	RttMdev          float32
	TimeMilliseconds uint32
	Transmitted      uint32
}

// Used for labeling values for display
const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
)

var (
	count         int
	getKube       bool
	localHostname string
	hostsString   string
	iperf3Args    []string
	pingArgs      []string
	iperf3Flags   string
	pingFlags     string
	kubeClientset *kubernetes.Clientset
	kubeConfig    string
	testsString   string
	mergeFiles    string
	output        string
	results       Results
	verbose       bool
)

func main() {
	var hostsString string
	flag.StringVar(&hostsString, "hosts", "internet:127.0.0.1", "List of test, host name & IP/hostname of each non-container host (<endpoint>:<ip|hostname>[:<port>]). Ex 'internet:6.6.6.6,worker1:7.7.7.7'")
	flag.StringVar(&testsString, "tests", "internet:netperf-lb-public,internet:netperf-np-public", "List of test, client:server pairs defining tests to run ([<iperf3>:]<client name>:<server name>>).")
	flag.StringVar(&output, "output", "-", "Named file for json results")
	flag.StringVar(&iperf3Flags, "iperf3Flags", "", "Flags to pass to iperf. This is above and beyond '-c'")
	flag.StringVar(&pingFlags, "pingFlags", "-f -c 10000 -s 1500 -l 4", "Flags to pass to ping.")
	flag.IntVar(&count, "count", 1, "Number of times each test should be run")
	flag.BoolVar(&verbose, "verbose", false, "verbose logging output")
	flag.BoolVar(&getKube, "kube", true, "Acquire netperf-* endpoints from kubernetes")
	flag.StringVar(&kubeConfig, "kubecfg", "", "Location of the kubernetes client configuration file. Default is to use KUBECONFIG.")
	flag.StringVar(&mergeFiles, "merge", "", "Merge the specified comma separated files into a single json file and summarize the results")
	flag.Parse()

	if mergeFiles == "" {
		ips := getLocalhostIPs()
		localHostname, _ = os.Hostname()
		if verbose {
			fmt.Println("localHostname:", localHostname)
		}

		// Initialize iperf3 arguments
		arguments := 8
		var flags []string
		if iperf3Flags != "" {
			flags = strings.Split(iperf3Flags, " ")
			arguments += len(flags)
		}

		iperf3Args = make([]string, arguments)
		//iperf3Args[HostArg] = <name of remote host>
		iperf3Args[1] = "iperf3"
		iperf3Args[2] = "-c"
		// iperf3Args[IPArg] = <ip of server>
		iperf3Args[4] = "-p"
		// iperf3Args[PortArg] = <port of server>
		iperf3Args[6] = "-J"

		index := 7
		for _, flag := range flags {
			iperf3Args[index] = flag
			index++
		}
		iperf3Args[index] = "--reverse"

		// Initialize ping arguments
		arguments = 4
		if pingFlags != "" {
			flags = strings.Split(pingFlags, " ")
			arguments += len(flags)
		}

		pingArgs = make([]string, arguments)
		//pingArgs[HostArg] = <name of remote host>
		pingArgs[1] = "sudo"
		pingArgs[2] = "ping"

		index = 3
		for _, flag := range flags {
			pingArgs[index] = flag
			index++
		}

		// Initialize Test definitions based on parameters
		var tests []Test
		if testsString != "" {
			hs := strings.Split(testsString, ",")
			testIndex := 0
			tests = make([]Test, len(hs)*count)
			for _, h := range hs {
				nameTest := strings.Split(h, ":")
				testType := Iperf3Test
				index := 0
				if len(nameTest) == 3 {
					switch nameTest[0] {
					case "iperf3":
					case "ping":
						testType = PingTest
					default:
						fmt.Println("ERROR: Test isn't supported:", nameTest[0])
						os.Exit(1)
					}
					index++
				}
				for k := 0; k < count; k++ {
					tests[testIndex] = Test{Client: nameTest[index], Server: nameTest[index+1], TestType: testType}
					testIndex++
				}
			}
		}

		// Create an Endpoint for each potential test server
		var endpoints map[string]Endpoint
		if getKube {
			getKubeClientSet()
			endpoints = getKubeServices()
		} else {
			endpoints = make(map[string]Endpoint)
		}

		if hostsString != "" {
			hs := strings.Split(hostsString, ",")
			for _, h := range hs {
				nameHost := strings.Split(h, ":")
				if _, ok := ips[nameHost[1]]; ok {
					nameHost[1] = localHostname
				}
				port := int32(5201)
				if len(nameHost) == 3 {
					aport, _ := strconv.ParseInt(nameHost[2], 10, 32)
					port = int32(aport)
				}
				var endpoint Endpoint
				endpoint = Endpoint{Name: nameHost[0], IP: nameHost[1], Port: port, Type: EndpointTypeHost}
				endpoints[endpoint.Name] = endpoint
			}
		}

		// Execute the tests
		results = Results{Endpoints: endpoints, Tests: tests}
		for i, test := range results.Tests {
			if verbose {
				fmt.Println("Executing test", test.TestType, "from:", test.Client, "to", test.Server)
			}
			if clientEndpoint, ok := endpoints[test.Client]; ok {
				if serverEndpoint, ok := endpoints[test.Server]; ok {
					switch test.TestType {
					case Iperf3Test:
						results.Tests[i] = executeIperf3(test, clientEndpoint, serverEndpoint)
					case PingTest:
						results.Tests[i] = executePing(test, clientEndpoint, serverEndpoint)
					default:
						panic(fmt.Sprintln("Test not supported: ", test.TestType))
					}
				} else {
					test.Error = fmt.Sprintf("ERROR: The server endpoint wasn't found: %s\n", test.Server)
					fmt.Fprintln(os.Stderr, test.Error)
				}
			} else {
				test.Error = fmt.Sprintf("ERROR: The cleint endpoint wasn't found: %s\n", test.Client)
				fmt.Fprintln(os.Stderr, test.Error)
			}
		}
	} else {
		// TODO Handle case where json files containe raw iperf3 output
		// Endpoints with same name will be merged. Thus the person merging must make sure that makes sense.
		endpoints := make(map[string]Endpoint)
		tests := make([]Test, 0, 100) // TODO Hard limits aren't good
		files := strings.Split(mergeFiles, ",")
		for _, f := range files {
			fmt.Println("Processing", f)
			// #nosec G304
			data, err := ioutil.ReadFile(f)
			if err != nil && err != io.EOF {
				fmt.Println(f, err.Error())
				continue
			}
			//fmt.Println("Openned file")
			//fmt.Println(data)
			var tmpResults Results
			err = json.Unmarshal(data, &tmpResults)
			if err != nil {
				fmt.Println("Unable to unmarshal results:", f, err.Error())
				continue
			}

			//fmt.Println("unmarshalled json")
			for _, e := range tmpResults.Endpoints {
				if _, ok := endpoints[e.Name]; ok == false {
					endpoints[e.Name] = e
				}
			}

			//fmt.Println("processed endpoints")
			for _, t := range tmpResults.Tests {
				if t.Error == "" {
					//tests[t.Name] = t
					tests = append(tests, t)
				}
			}
			//fmt.Println("processed tests")
		}
		results = Results{Endpoints: endpoints, Tests: tests}
	}

	if verbose {
		fmt.Println("Processing results")
	}

	// Output the results
	jsonData, err := json.Marshal(results)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	if output == "-" {
		os.Stdout.Write(jsonData)
		fmt.Fprintln(os.Stderr)
	} else {
		err = ioutil.WriteFile(output, jsonData, 0644)
		sumarizeResults(results)
	}
}

func sumarizeResults(results Results) {

	collated := make(map[string][]Test)
	for _, test := range results.Tests {
		id := fmt.Sprintf("%s-%s-%s-", test.TestType, test.Client, test.Server)
		if _, ok := collated[id]; ok == false {
			collated[id] = make([]Test, 0, 3)
		}
		// TODO highlight errors and don't include in collated results
		collated[id] = append(collated[id], test)
	}
	for _, co := range collated {
		switch co[0].TestType {
		case Iperf3Test:
			sumarizeIperf3Results(co)
		case PingTest:
			sumarizePingResults(co)
		default:
			panic(fmt.Sprintln("Test not supported: ", co[0].TestType))
		}
	}
}

func sumarizeIperf3Results(co []Test) {
	var bytes uint64
	var seconds float64

	for _, test := range co {
		bytes += test.Iperf3.End.SumSent.Bytes
		seconds += test.Iperf3.End.SumSent.Seconds
		test.Time = test.Iperf3.Start.Timestamp.Time
	}
	fmt.Printf("iperf3 Result: %s, %s -> %s sent %s, avg rate %s\n", co[0].Time, co[0].Client, co[0].Server, byteLabel(bytes), rateLabel(float64(bytes*8)/seconds))
}

func sumarizePingResults(co []Test) {
	var transmitted uint64
	var received uint64
	var rttMin float32 = math.MaxFloat32
	var rttAvg float32
	var rttMax float32
	var maxRttMdev float32
	var tests uint

	for _, test := range co {
		if verbose {
			fmt.Println(test)
		}
		if test.Error == "" {
			transmitted += uint64(test.Ping.Transmitted)
			received += uint64(test.Ping.Received)
			if rttMin > test.Ping.RttMin {
				rttMin = test.Ping.RttMin
			}
			rttAvg += test.Ping.RttAvg
			if rttMax < test.Ping.RttMax {
				rttMax = test.Ping.RttMax
			}
			if maxRttMdev < test.Ping.RttMdev {
				maxRttMdev = test.Ping.RttMdev
			}
			tests++
		}
	}
	if tests > 0 {
		rttAvg = rttAvg / float32(tests)
		fmt.Printf("Ping Result: %s, %s -> %s sent %d,  received %d, loss %d, rttMin %0.3f, rttAvg %0.3f, rttMax %0.3f max rttMdev %0.3f\n", "", co[0].Client, co[0].Server, transmitted, received, transmitted-received, rttMin, rttAvg, rttMax, maxRttMdev)
	}
}

func getLocalhostIPs() map[string]string {
	ips := make(map[string]string)
	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		for _, addr := range addrs {
			ipString := addr.String()
			if strings.Contains(ipString, ":") == false {
				s := strings.Split(ipString, "/")
				ips[s[0]] = s[0]
			}
		}
	}
	ips["localhost"] = "localhost"
	return ips
}

func dumpJSON(obj interface{}) {
	data, err := json.Marshal(obj)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	os.Stdout.Write(data)
	fmt.Fprintln(os.Stderr)
}

func getKubeServices() map[string]Endpoint {

	var endpoints = make(map[string]Endpoint)

	if kubeClientset != nil {
		var listOpts = metav1.ListOptions{LabelSelector: "app in (netperf-ci, netperf-np, netperf-lb, netperf-ingress)"}
		svcs, err := kubeClientset.CoreV1().Services(k8sv1.NamespaceDefault).List(context.TODO(), listOpts)

		if err != nil {
			fmt.Fprintln(os.Stderr, "FAILED: ", err)
		}

		for _, s := range svcs.Items {
			if verbose {
				fmt.Println("name: ", s.Name)
				fmt.Println("type: ", s.Spec.Type)
				fmt.Println("clusterIP: ", s.Spec.ClusterIP)
				if s.Spec.Type == k8sv1.ServiceTypeLoadBalancer {
					fmt.Println("LB: ", s.Status.LoadBalancer.Ingress[0].IP)
				}
				fmt.Println("ports: ", s.Spec.Ports)
				for _, p := range s.Spec.Ports {
					fmt.Println("    Name: ", p.Name)
					fmt.Println("    Port: ", p.Port)
					fmt.Println("    NodePort: ", p.NodePort)
					fmt.Println("    TargetPort: ", p.TargetPort)
					fmt.Println("")
				}
				fmt.Println("")
			}

			var pods []Pod
			if len(s.Spec.Selector) > 0 {
				for name, selector := range s.Spec.Selector {
					selector = name + " in (" + selector + ")"
					pods = getServiceNodes(selector)
				}
			} else {
				fmt.Fprintln(os.Stderr, "WARN: No pod selector for service: ", s.Name)
			}
			var endpoint = Endpoint{Name: s.Name, Type: converEndpointTypes(s.Spec.Type), Pods: pods}

			switch endpoint.Type {
			case EndpointTypeClusterIP:
				// TODO fmt.Println("CloudIP not supported but needs to be")
				continue
			case EndpointTypeLoadBalancer:
				endpoint.IP = s.Status.LoadBalancer.Ingress[0].IP
				endpoint.Port = getPort(s.Spec.Ports, "iperf").Port
			case EndpointTypeNodePort:
				if len(endpoint.Pods) > 0 {
					endpoint.IP = endpoint.Pods[0].NodeIP
				} else {
					fmt.Fprintln(os.Stderr, "WARN: No Node IP defined for NodePort service ", endpoint.Name)
				}
				endpoint.Port = getPort(s.Spec.Ports, "iperf").NodePort
			default:
				fmt.Println("ERROR: Port type isn't supported: ", endpoint.Type)
				continue
			}

			endpoints[endpoint.Name] = endpoint
		}
	}
	return endpoints
}

func getServiceNodes(selector string) []Pod {

	// Select nodes where netperf is running
	var pods []Pod
	if kubeClientset != nil {
		var listOpts = metav1.ListOptions{LabelSelector: selector}
		podList, err := kubeClientset.CoreV1().Pods("default").List(context.TODO(), listOpts)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERROR: ", err)
		}
		pods = make([]Pod, len(podList.Items))
		for i, p := range podList.Items {
			pods[i].Name = p.Name
			pods[i].NodeIP = p.Status.HostIP
		}
	}
	return pods
}

func executeIperf3(test Test, client Endpoint, server Endpoint) Test {
	if verbose {
		fmt.Println("executIperf3() service:", server.Name, server.Type, server.IP, server.Port)
	}

	iperf3Args[IPArg] = server.IP
	iperf3Args[PortArg] = strconv.FormatInt(int64(server.Port), 10)

	var testArgs []string
	var command string
	if client.IP == localHostname || server.IP == localHostname {
		if testPort(server.IP, server.Port) == false {
			test.Error = "Port isn't active"
			return test
		}

		if server.IP == localHostname {
			// Use '--reverse' when service == localhost
			iperf3Args[IPArg] = client.IP
			testArgs = iperf3Args[2:]
		} else {
			testArgs = iperf3Args[2 : len(iperf3Args)-1]
		}
		command = "iperf3"
	} else {
		// This approach requires that users ~/.ssh/config file be setup so that
		// the user can access the machine with "ssh <hostname>"
		command = "ssh"
		testArgs = iperf3Args[:len(iperf3Args)-1]
		iperf3Args[HostArg] = client.IP
	}

	if verbose {
		fmt.Println("Running", command, testArgs)
	}
	// #nosec G204
	data, err := exec.Command(command, testArgs...).Output()
	if err != nil {
		test.Error = err.Error()
		return test
	}

	var iperf3 Iperf3
	err = json.Unmarshal(data, &iperf3)
	if err != nil {
		test.Error = fmt.Sprintln("Unable to unmarshal iperf3 results:", err.Error())
		return test
	}

	if iperf3.Error == "" && (output != "-" || verbose) {
		test.Time = iperf3.Start.Timestamp.Time
		//fmt.Printf("Result: %s, %s -> %s sent %s, rate %s\n", test.Time, client.Name, server.Name, byteLabel(iperf3.End.SumSent.Bytes), rateLabel(iperf3.End.SumSent.BitsPerSecond))
	} else {
		test.Error = iperf3.Error
	}
	test.Iperf3 = iperf3

	return test
}

func executePing(test Test, client Endpoint, server Endpoint) Test {
	if verbose {
		fmt.Println("executPing() service: ", server.Name, server.Type, server.IP, server.Port)
	}

	var ping Ping
	pingArgs[len(pingArgs)-1] = server.IP

	var testArgs []string
	var command string
	if client.IP == localHostname {
		testArgs = pingArgs[2:]
		command = "sudo"
	} else {
		// This approach requires that users ~/.ssh/config file be setup so that
		// the user can access the machine with "ssh <hostname>"
		command = "ssh"
		testArgs = pingArgs
		pingArgs[HostArg] = client.IP
	}

	if verbose {
		fmt.Println("Running", command, testArgs)
	}
	// #nosec G204
	data, err := exec.Command(command, testArgs...).Output()
	if err != nil {
		test.Error = err.Error()
		if verbose {
			fmt.Println("Error: Executing ping test:", err.Error())
		}
		return test
	}

	// PING 169.55.193.85 (169.55.193.85) 1500(1528) bytes of data.
	//
	// --- 169.55.193.85 ping statistics ---
	// 1000 packets transmitted, 1000 received, 0% packet loss, time 6454ms
	// rtt min/avg/max/mdev = 25.686/25.872/26.545/0.090 ms, pipe 6, ipg/ewma 6.460/25.858 ms

	s := string(data)

	// OSx
	// var packetRegex = regexp.MustCompile(`(?P<transmit>\d+) packets transmitted, (?P<receive>\d+) (?:packets )?received, (?P<loss>\d+\.?\d*)% packet loss, time (?P<time>\d+)ms`)
	// var rttRegex = regexp.MustCompile(`rtt min/avg/max/mdev = (?P<rttMin>\d+\.?\d*)/(?P<rttAvg>\d+\.?\d*)/(?P<rttMax>\d+\.?\d*)/(?P<rttMdev>\d+\.?\d*) ms, pipe (\d+), ipg/ewma (?P<ipg>\d+\.?\d*)/(?P<ewma>\d+\.?\d*) ms`)
	var packetRegex = regexp.MustCompile(`(?P<transmit>\d+) packets transmitted, (?P<receive>\d+) (?:packets )?received, (?P<loss>\d+\.?\d*)% packet loss`)
	var rttRegex = regexp.MustCompile(`min/avg/max/(?:[mdevstddev]+) = (?P<rttMin>\d+\.?\d*)/(?P<rttAvg>\d+\.?\d*)/(?P<rttMax>\d+\.?\d*)/(?P<rttMdev>\d+\.?\d*) ms`)

	packetMatch := packetRegex.FindStringSubmatch(s)
	rttMatch := rttRegex.FindStringSubmatch(s)
	if len(packetMatch) <= 1 || len(rttMatch) <= 1 {
		if verbose {
			fmt.Println(s)
		}
		test.Error = "Error: Results of ping couldn't be parsed. It is likely ping wasn't successful"
		return test
	}

	result := make(map[string]string)
	for i, name := range packetRegex.SubexpNames() {
		if i != 0 {
			result[name] = packetMatch[i]
		}
	}

	for i, name := range rttRegex.SubexpNames() {
		if i != 0 {
			result[name] = rttMatch[i]
		}
	}
	if verbose {
		fmt.Println("Ping results", result)
	}

	// TODO Hangle if some of conversions don't happen. IE should error be set?
	flt, err := strconv.ParseFloat(result["rttMin"], 32)
	if err == nil {
		ping.RttMin = float32(flt)
	}
	flt, err = strconv.ParseFloat(result["rttAvg"], 32)
	if err == nil {
		ping.RttAvg = float32(flt)
	}
	flt, err = strconv.ParseFloat(result["rttMax"], 32)
	if err == nil {
		ping.RttMax = float32(flt)
	}
	flt, err = strconv.ParseFloat(result["rttMdev"], 32)
	if err == nil {
		ping.RttMdev = float32(flt)
	}
	intRes, err := strconv.ParseInt(result["receive"], 10, 32)
	if err == nil {
		ping.Received = uint32(intRes)
	}
	intRes, err = strconv.ParseInt(result["transmit"], 10, 32)
	if err == nil {
		ping.Transmitted = uint32(intRes)
	}

	test.Ping = ping

	return test
}

// testPort checks to see that there is a server handling connections on port from localhost
func testPort(host string, port int32) bool {
	var success bool
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	if err == nil {
		success = true
		conn.Close()
	}
	return success
}

func byteLabel(bytes uint64) string {
	label := ""
	switch {
	case bytes > GB:
		label = fmt.Sprintf("%0.1f GB", float64(bytes)/GB)
	case bytes > MB:
		label = fmt.Sprintf("%0.1f MB", float64(bytes)/MB)
	case bytes > KB:
		label = fmt.Sprintf("%0.1f KB", float64(bytes)/KB)
	default:
		label = fmt.Sprintf("%d Bytes", bytes)
	}
	return label
}

func rateLabel(rate float64) string {
	label := ""
	switch {
	case rate > GB:
		label = fmt.Sprintf("%0.1f Gbits/sec", rate/GB)
	case rate > MB:
		label = fmt.Sprintf("%0.1f Mbits/sec", rate/MB)
	case rate > KB:
		label = fmt.Sprintf("%0.1f Kbits/sec", rate/KB)
	default:
		label = fmt.Sprintf("%0.1f bits/sec", rate)
	}
	return label
}

func getPort(ports []k8sv1.ServicePort, name string) k8sv1.ServicePort {
	for _, port := range ports {
		if port.Name == name {
			return port
		}
	}
	fmt.Fprintln(os.Stderr, "Error: Port not found. Port name: ", name)
	// TODO harden callers for this condition
	os.Exit(1)

	var result k8sv1.ServicePort
	return result
}

func converEndpointTypes(k8sEndpointType k8sv1.ServiceType) EndpointType {
	var endpointType EndpointType
	switch k8sEndpointType {
	case k8sv1.ServiceTypeClusterIP:
		endpointType = EndpointTypeClusterIP
	case k8sv1.ServiceTypeNodePort:
		endpointType = EndpointTypeNodePort
	case k8sv1.ServiceTypeLoadBalancer:
		endpointType = EndpointTypeLoadBalancer
	default:
		fmt.Fprintln(os.Stderr, "Error: k8sv1.ServiceType not supported: ", k8sEndpointType)
	}
	return endpointType
}

/* stolen from api/armada-perf-client/lib/cluster/cluster.go
 * TODO need to break out a new kubernetes library?
 */
// getKubeClientSet loads the k8s config for the cluster
func getKubeClientSet() *kubernetes.Clientset {
	if verbose {
		fmt.Println("getKubeClientSet")
	}

	if kubeConfig == "" {
		kubeConfig = os.Getenv("KUBECONFIG")
	}

	if kubeConfig != "" {
		if verbose {
			fmt.Println("Getting kube config")
		}
		config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		clientset, _ := kubernetes.NewForConfig(config)
		kubeClientset = clientset
	} else {
		fmt.Fprintln(os.Stderr, "Error: Kube configuration isn't defined. Set KUBECONFIG or specify as parameter")
		os.Exit(1)
	}

	return kubeClientset
}
