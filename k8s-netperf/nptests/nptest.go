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
 nptest.go

 Dual-mode program - runs as both the orchestrator and as the worker nodes depending on command line flags
 The RPC API is contained wholly within this file.

 Based on https://github.com/kubernetes/perf-tests/blob/master/network/benchmarks/netperf/nptest/nptest.go
 With some tweaks for armada-performance testing
*/

package main

// Imports only base Golang packages
import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"sync"
	"time"
)

type point struct {
	mss       int
	bandwidth string
	index     int
}

var mode string
var port string
var host string
var worker string
var kubenode string
var podname string

var workerStateMap map[string]*workerState

var iperfTCPOutputRegexp *regexp.Regexp
var iperfUDPOutputRegexp *regexp.Regexp
var iperfCPUOutputRegexp *regexp.Regexp

var dataPoints map[string][]point
var dataPointKeys []string
var datapointsFlushed bool

var globalLock sync.Mutex

const (
	workerMode        = "worker"
	orchestratorMode  = "orchestrator"
	iperf3Path        = "/usr/bin/iperf3"
	outputCaptureFile = "/tmp/output.txt"
	// Setting mssMin to 0 means it will run without MSS setting
	mssMin               = 0
	mssMax               = 1460
	mssStepSize          = 480
	parallelStreams      = "8"
	rpcServicePort       = "5202"
	localhostIPv4Address = "127.0.0.1"
)

const (
	iperfTCPTest = iota
	iperfUDPTest = iota
	netperfTest  = iota
)

// NetPerfRPC service that exposes RegisterClient and ReceiveOutput for clients
type NetPerfRPC int

// ClientRegistrationData stores a data about a single client
type ClientRegistrationData struct {
	Host     string
	KubeNode string
	Worker   string
	IP       string
}

// IperfClientWorkItem represents a single task for an Iperf client
type IperfClientWorkItem struct {
	Host string
	Port string
	MSS  int
	Type int
}

// IperfServerWorkItem represents a single task for an Iperf server
type IperfServerWorkItem struct {
	ListenPort string
	Timeout    int
}

// WorkItem represents a single task for a worker
type WorkItem struct {
	IsClientItem bool
	IsServerItem bool
	IsIdle       bool
	ClientItem   IperfClientWorkItem
	ServerItem   IperfServerWorkItem
}

type workerState struct {
	sentServerItem bool
	idle           bool
	IP             string
	worker         string
}

// WorkerOutput stores the results from a single worker
type WorkerOutput struct {
	Output string
	Code   int
	Worker string
	Type   int
}

type testcase struct {
	SourceNode      string
	DestinationNode string
	Label           string
	ClusterIP       bool
	Finished        bool
	MSS             int
	Type            int
}

var testcases []*testcase
var currentJobIndex int

func init() {
	flag.StringVar(&mode, "mode", "worker", "Mode for the daemon (worker | orchestrator)")
	flag.StringVar(&port, "port", rpcServicePort, "Port to listen on (defaults to 5202)")
	flag.StringVar(&host, "host", "", "IP address to bind to (defaults to 0.0.0.0)")

	workerStateMap = make(map[string]*workerState)
	testcases = []*testcase{
		{SourceNode: "netperf-w1", DestinationNode: "netperf-w2", Label: "1 iperf TCP. Same VM using Pod IP", Type: iperfTCPTest, ClusterIP: false, MSS: mssMin},
		{SourceNode: "netperf-w1", DestinationNode: "netperf-w2", Label: "2 iperf TCP. Same VM using Virtual IP", Type: iperfTCPTest, ClusterIP: true, MSS: mssMin},
		{SourceNode: "netperf-w1", DestinationNode: "netperf-w3", Label: "3 iperf TCP. Remote VM using Pod IP", Type: iperfTCPTest, ClusterIP: false, MSS: mssMin},
		{SourceNode: "netperf-w3", DestinationNode: "netperf-w2", Label: "4 iperf TCP. Remote VM using Virtual IP", Type: iperfTCPTest, ClusterIP: true, MSS: mssMin},
		{SourceNode: "netperf-w2", DestinationNode: "netperf-w2", Label: "5 iperf TCP. Hairpin Pod to own Virtual IP", Type: iperfTCPTest, ClusterIP: true, MSS: mssMin},
		{SourceNode: "netperf-w1", DestinationNode: "netperf-w2", Label: "6 iperf UDP. Same VM using Pod IP", Type: iperfUDPTest, ClusterIP: false, MSS: mssMax},
		{SourceNode: "netperf-w1", DestinationNode: "netperf-w2", Label: "7 iperf UDP. Same VM using Virtual IP", Type: iperfUDPTest, ClusterIP: true, MSS: mssMax},
		{SourceNode: "netperf-w1", DestinationNode: "netperf-w3", Label: "8 iperf UDP. Remote VM using Pod IP", Type: iperfUDPTest, ClusterIP: false, MSS: mssMax},
		{SourceNode: "netperf-w3", DestinationNode: "netperf-w2", Label: "9 iperf UDP. Remote VM using Virtual IP", Type: iperfUDPTest, ClusterIP: true, MSS: mssMax},
	}

	currentJobIndex = 0

	// Regexes to parse the Mbits/sec out of iperf TCP, UDP and netperf output
	iperfTCPOutputRegexp = regexp.MustCompile("SUM.*\\s+(\\d+)\\sMbits/sec\\s+receiver")
	iperfUDPOutputRegexp = regexp.MustCompile("\\s+(\\S+)\\sMbits/sec\\s+\\S+\\s+ms\\s+")
	iperfCPUOutputRegexp = regexp.MustCompile(`local/sender\s(\d+\.\d+)%\s\((\d+\.\d+)%\w/(\d+\.\d+)%\w\),\sremote/receiver\s(\d+\.\d+)%\s\((\d+\.\d+)%\w/(\d+\.\d+)%\w\)`)

	dataPoints = make(map[string][]point)
}

func initializeOutputFiles() {
	fd, err := os.OpenFile(outputCaptureFile, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		fmt.Println("Failed to open output capture file", err)
		os.Exit(2)
	}
	fd.Close()
}

func main() {
	initializeOutputFiles()
	flag.Parse()
	if !validateParams() {
		fmt.Println("Failed to parse cmdline args - fatal error - bailing out")
		os.Exit(1)

	}
	grabEnv()
	fmt.Println("Running as", mode, "...")
	if mode == orchestratorMode {
		orchestrate()
	} else {
		startWork()
	}
	fmt.Println("Terminating npd")
}

func grabEnv() {
	worker = os.Getenv("worker")
	kubenode = os.Getenv("kubenode")
	podname = os.Getenv("HOSTNAME")
}

func validateParams() (rv bool) {
	rv = true
	if mode != workerMode && mode != orchestratorMode {
		fmt.Println("Invalid mode", mode)
		return false
	}

	if len(port) == 0 {
		fmt.Println("Invalid port", port)
		return false
	}

	if (len(host)) == 0 {
		if mode == orchestratorMode {
			host = os.Getenv("NETPERF_ORCH_SERVICE_HOST")
		} else {
			host = os.Getenv("NETPERF_ORCH_SERVICE_HOST")
		}
	}
	return
}

func allWorkersIdle() bool {
	for _, v := range workerStateMap {
		if !v.idle {
			return false
		}
	}
	return true
}

func getWorkerPodIP(worker string) string {
	return workerStateMap[worker].IP
}

func allocateWorkToClient(workerS *workerState, reply *WorkItem) {
	if !allWorkersIdle() {
		reply.IsIdle = true
		return
	}

	// System is all idle - pick up next work item to allocate to client
	for n, v := range testcases {
		if v.Finished {
			continue
		}
		if v.SourceNode != workerS.worker {
			reply.IsIdle = true
			return
		}
		if _, ok := workerStateMap[v.DestinationNode]; !ok {
			reply.IsIdle = true
			return
		}
		fmt.Printf("Requesting jobrun '%s' from %s to %s for MSS %d\n", v.Label, v.SourceNode, v.DestinationNode, v.MSS)
		reply.ClientItem.Type = v.Type
		reply.IsClientItem = true
		workerS.idle = false
		currentJobIndex = n

		if !v.ClusterIP {
			reply.ClientItem.Host = getWorkerPodIP(v.DestinationNode)
		} else {
			reply.ClientItem.Host = os.Getenv("NETPERF_W2_SERVICE_HOST")
		}

		switch {
		case v.Type == iperfTCPTest || v.Type == iperfUDPTest:
			reply.ClientItem.Port = "5201"
			reply.ClientItem.MSS = v.MSS

			v.MSS = v.MSS + mssStepSize
			if v.MSS > mssMax {
				v.Finished = true
			}
			return
		}
	}

	for _, v := range testcases {
		if !v.Finished {
			return
		}
	}

	if !datapointsFlushed {
		fmt.Println("ALL TESTCASES AND MSS RANGES COMPLETE - GENERATING CSV OUTPUT")
		flushDataPointsToCsv()
		datapointsFlushed = true
	}

	reply.IsIdle = true
}

// RegisterClient registers a single and assign a work item to it
func (t *NetPerfRPC) RegisterClient(data *ClientRegistrationData, reply *WorkItem) error {
	globalLock.Lock()
	defer globalLock.Unlock()

	state, ok := workerStateMap[data.Worker]

	if !ok {
		// For new clients, trigger an iperf server start immediately
		state = &workerState{sentServerItem: true, idle: true, IP: data.IP, worker: data.Worker}
		workerStateMap[data.Worker] = state
		reply.IsServerItem = true
		reply.ServerItem.ListenPort = "5201"
		reply.ServerItem.Timeout = 3600
		return nil
	}

	// Worker defaults to idle unless the allocateWork routine below assigns an item
	state.idle = true

	// Give the worker a new work item or let it idle loop another 5 seconds
	allocateWorkToClient(state, reply)
	return nil
}

func writeOutputFile(filename, data string) {
	fd, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Println("Failed to append to existing file", filename, err)
		return
	}
	defer fd.Close()

	if _, err = fd.WriteString(data); err != nil {
		fmt.Println("Failed to append to existing file", filename, err)
	}
}

func registerDataPoint(label string, mss int, value string, index int) {
	if sl, ok := dataPoints[label]; !ok {
		dataPoints[label] = []point{{mss: mss, bandwidth: value, index: index}}
		dataPointKeys = append(dataPointKeys, label)
	} else {
		dataPoints[label] = append(sl, point{mss: mss, bandwidth: value, index: index})
	}
}

func flushDataPointsToCsv() {
	var buffer string

	// Write the MSS points for the X-axis before dumping all the testcase datapoints
	for _, points := range dataPoints {
		if len(points) == 1 {
			continue
		}
		buffer = fmt.Sprintf("%-45s, Maximum,", "MSS")
		for _, p := range points {
			buffer = buffer + fmt.Sprintf(" %d,", p.mss)
		}
		break
	}
	fmt.Println(buffer)

	for _, label := range dataPointKeys {
		buffer = fmt.Sprintf("%-45s,", label)
		points := dataPoints[label]
		var max float64
		for _, p := range points {
			fv, _ := strconv.ParseFloat(p.bandwidth, 64)
			if fv > max {
				max = fv
			}
		}
		buffer = buffer + fmt.Sprintf("%f,", max)
		for _, p := range points {
			buffer = buffer + fmt.Sprintf("%s,", p.bandwidth)
		}
		fmt.Println(buffer)
	}
	fmt.Println("END CSV DATA")
}

func parseIperfTCPBandwidth(output string) string {
	// Parses the output of iperf3 and grabs the group Mbits/sec from the output
	match := iperfTCPOutputRegexp.FindStringSubmatch(output)
	if match != nil && len(match) > 1 {
		return match[1]
	}
	return "0"
}

func parseIperfUDPBandwidth(output string) string {
	// Parses the output of iperf3 (UDP mode) and grabs the Mbits/sec from the output
	match := iperfUDPOutputRegexp.FindStringSubmatch(output)
	if match != nil && len(match) > 1 {
		return match[1]
	}
	return "0"
}

func parseIperfCPUUsage(output string) (string, string) {
	// Parses the output of iperf and grabs the CPU usage on sender and receiver side from the output
	match := iperfCPUOutputRegexp.FindStringSubmatch(output)
	if match != nil && len(match) > 1 {
		return match[1], match[4]
	}
	return "0", "0"
}

// ReceiveOutput processes a data received from a single client
func (t *NetPerfRPC) ReceiveOutput(data *WorkerOutput, reply *int) error {
	globalLock.Lock()
	defer globalLock.Unlock()

	testcase := testcases[currentJobIndex]

	var outputLog string
	var bw string
	var cpuSender string
	var cpuReceiver string

	switch data.Type {
	case iperfTCPTest:
		mss := testcases[currentJobIndex].MSS - mssStepSize
		if mss < 0 {
			mss = 0
		}
		outputLog = outputLog + fmt.Sprintln("Received TCP output from worker", data.Worker, "for test", testcase.Label,
			"from", testcase.SourceNode, "to", testcase.DestinationNode, "MSS:", mss) + data.Output
		writeOutputFile(outputCaptureFile, outputLog)
		bw = parseIperfTCPBandwidth(data.Output)
		cpuSender, cpuReceiver = parseIperfCPUUsage(data.Output)
		registerDataPoint(testcase.Label, mss, bw, currentJobIndex)

	case iperfUDPTest:
		mss := testcases[currentJobIndex].MSS - mssStepSize
		outputLog = outputLog + fmt.Sprintln("Received UDP output from worker", data.Worker, "for test", testcase.Label,
			"from", testcase.SourceNode, "to", testcase.DestinationNode, "MSS:", mss) + data.Output
		writeOutputFile(outputCaptureFile, outputLog)
		bw = parseIperfUDPBandwidth(data.Output)
		registerDataPoint(testcase.Label, mss, bw, currentJobIndex)
	}

	switch data.Type {
	case iperfTCPTest:
		fmt.Println("Jobdone from worker", data.Worker, "Bandwidth was", bw, "Mbits/sec. CPU usage sender was", cpuSender, "%. CPU usage receiver was", cpuReceiver, "%.")
	default:
		fmt.Println("Jobdone from worker", data.Worker, "Bandwidth was", bw, "Mbits/sec")
	}

	return nil
}

func serveRPCRequests(port string) {
	baseObject := new(NetPerfRPC)
	rpc.Register(baseObject)
	rpc.HandleHTTP()
	listener, e := net.Listen("tcp", ":"+port)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	http.Serve(listener, nil)
}

// Blocking RPC server start - only runs on the orchestrator
func orchestrate() {
	serveRPCRequests(rpcServicePort)
}

// Walk the list of interfaces and find the first interface that has a valid IP
// Inside a container, there should be only one IP-enabled interface
func getMyIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return localhostIPv4Address
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback == 0 {
			addrs, _ := iface.Addrs()
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				return ip.String()
			}
		}
	}
	return "127.0.0.1"
}

func handleClientWorkItem(client *rpc.Client, workItem *WorkItem) {
	fmt.Println("Orchestrator requests worker run item Type:", workItem.ClientItem.Type)
	switch {
	case workItem.ClientItem.Type == iperfTCPTest || workItem.ClientItem.Type == iperfUDPTest:
		outputString := iperfClient(workItem.ClientItem.Host, workItem.ClientItem.Port, workItem.ClientItem.MSS, workItem.ClientItem.Type)
		var reply int
		client.Call("NetPerfRPC.ReceiveOutput", WorkerOutput{Output: outputString, Worker: worker, Type: workItem.ClientItem.Type}, &reply)
	}
	// Client COOLDOWN period before asking for next work item to replenish burst allowance policers etc
	time.Sleep(10 * time.Second)
}

// startWork : Entry point to the worker infinite loop
func startWork() {
	for true {
		var timeout time.Duration
		var client *rpc.Client
		var err error

		timeout = 5
		for true {
			fmt.Println("Attempting to connect to orchestrator at", host)
			client, err = rpc.DialHTTP("tcp", host+":"+port)
			if err == nil {
				break
			}
			fmt.Println("RPC connection to ", host, " failed:", err)
			time.Sleep(timeout * time.Second)
		}

		for true {
			clientData := ClientRegistrationData{Host: podname, KubeNode: kubenode, Worker: worker, IP: getMyIP()}
			var workItem WorkItem

			if err := client.Call("NetPerfRPC.RegisterClient", clientData, &workItem); err != nil {
				// RPC server has probably gone away - attempt to reconnect
				fmt.Println("Error attempting RPC call", err)
				break
			}

			switch {
			case workItem.IsIdle == true:
				time.Sleep(5 * time.Second)
				continue

			case workItem.IsServerItem == true:
				fmt.Println("Orchestrator requests worker run iperf server")
				go iperfServer()
				time.Sleep(1 * time.Second)

			case workItem.IsClientItem == true:
				handleClientWorkItem(client, &workItem)
			}
		}
	}
}

// Invoke and indefinitely run an iperf server
func iperfServer() {
	output, success := cmdExec(iperf3Path, []string{iperf3Path, "-s", host, "-J", "-i", "60"}, 15)
	if success {
		fmt.Println(output)
	}
}

// Invoke and run an iperf client and return the output if successful.
func iperfClient(serverHost, serverPort string, mss int, workItemType int) (rv string) {
	var args []string
	var output string
	var success bool
	switch {
	case workItemType == iperfTCPTest:

		if mss > 0 {
			args = []string{iperf3Path, "-c", serverHost, "-V", "-N", "-i", "30", "-t", "10", "-f", "m", "-Z", "-P", parallelStreams, "-M", strconv.Itoa(mss)}
			output, success = cmdExec(iperf3Path, args, 15)
		} else {
			// Run without MSS setting
			args = []string{iperf3Path, "-c", serverHost, "-V", "-N", "-i", "30", "-t", "10", "-f", "m", "-Z", "-P", parallelStreams}
			output, success = cmdExec(iperf3Path, args, 15)
		}
		if success {
			rv = output
		}

	case workItemType == iperfUDPTest:
		args = []string{iperf3Path, "-c", serverHost, "-i", "30", "-t", "10", "-f", "m", "-b", "0", "-u"}
		output, success = cmdExec(iperf3Path, args, 15)
		// Add in a sleep otherwise UDP tests can affect tests that follow
		fmt.Println("Sleeping for 20 seconds between UDP tests to allow data time to flush")
		time.Sleep(20 * time.Second)
		if success {
			rv = output
		}
	}
	fmt.Println("Completed test with args: ", args, " , dumping logs below:")
	fmt.Println(output)
	return
}

func cmdExec(command string, args []string, timeout int32) (rv string, rc bool) {
	cmd := exec.Cmd{Path: command, Args: args}

	var stdoutput bytes.Buffer
	var stderror bytes.Buffer
	cmd.Stdout = &stdoutput
	cmd.Stderr = &stderror
	if err := cmd.Run(); err != nil {
		outputstr := stdoutput.String()
		errstr := stderror.String()
		fmt.Println("Failed to run", outputstr, "error:", errstr, err)
		return
	}

	rv = stdoutput.String()
	rc = true
	return
}
