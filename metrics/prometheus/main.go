/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2017, 2019 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"time"

	papi "github.com/prometheus/client_golang/api"
	prom "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"golang.org/x/net/context"
)

var logger *log.Logger

const debug bool = false

// typedefs for data holders; values indexed by only instance (hostname) or by instance & device

// SingleMap ....
type SingleMap map[string]float64

// DoubleMap ...
type DoubleMap map[string]map[string]float64

func init() {
	logger = log.New(os.Stdout, "metrics ", log.Lshortfile|log.Ltime)
}

func main() {
	var startTime, endTime time.Time
	var err error

	// default to retreiving an hour of data from the current time
	d, err := time.ParseDuration("-60m")

	if err != nil {
		logger.Fatalln("invalid duration", err)
	}

	endpoint := "http://localhost:9090"
	now := time.Now()
	startTime = now.Add(d)
	endTime = now

	if len(os.Args) > 1 {
		endpoint = os.Args[1]
	}
	// second argument must be a time
	if len(os.Args) > 2 {
		startTime, err = parseTime(now, os.Args[2])

		if err != nil {
			logger.Fatalln("invalid start time", os.Args[2])
		}
	}
	// third argument can be a time or a duration
	// duration can be negative which makes the first argument the end time
	if len(os.Args) > 3 {
		endTime, err = parseTime(now, os.Args[3])

		if err != nil {
			if debug {
				logger.Println("parsing", os.Args[3], "as a duration")
			}
			d, err = time.ParseDuration(os.Args[3])

			if err != nil {
				logger.Fatalln("invalid duration ", err)
			}

			endTime = startTime.Add(d)
		}
	}

	// swap times if necessary
	if endTime.Before(startTime) {
		startTime, endTime = endTime, startTime
	}

	fmt.Println("getting Prometheus data for", endpoint, "from", startTime.Format(time.RFC822), "to", endTime.Format(time.RFC822))

	// prometheus.Client
	client, err := papi.NewClient(papi.Config{Address: endpoint})

	if err != nil {
		logger.Println("error", err)
		return
	}

	// prometheus.QueryAPI
	q := prom.NewAPI(client)

	// Range used for all queries
	// note Step of 10 must line up with scrape interval to ensure all data is collected
	r := prom.Range{Start: startTime, End: endTime, Step: 10 * time.Second}

	// set to track all instances (hostnames)
	instances := make(map[string]struct{})

	// queries return model.Value which needs to be cast to String, Scalar, Vector or Matrix based on v.Type()
	fmt.Println("current CPU")
	outputSingleValues(&q, "100 * (1 - avg by(instance)(irate(node_cpu{mode='idle'}[30s])))", now)
	fmt.Println()

	cpuData := collectForSingleValue(&q, "100 * (1 - avg by(instance)(irate(node_cpu{mode='idle'}[30s])))", &r, instances)
	memoryData := collectForSingleValue(&q, "node_memory_Active / 1024 / 1024 / 1024", &r, instances)
	diskReadData := collectForTwoValues(&q, "(sum(irate(node_disk_bytes_read[30s])) by (instance,device)) / 1024 / 1024", &r, instances)
	diskWriteData := collectForTwoValues(&q, "(sum(irate(node_disk_bytes_written[30s])) by (instance,device)) / 1024 / 1024", &r, instances)
	networkReadData := collectForTwoValues(&q, "(irate(node_network_receive_bytes{device=~'^bond.*'}[30s])) / 1024 / 1024", &r, instances)
	networkWriteData := collectForTwoValues(&q, "(irate(node_network_transmit_bytes{device=~'^bond.*'}[30s])) / 1024 / 1024", &r, instances)

	// for consistent output, sort the instances, disks and network interfaces
	sortedInstances := make([]string, len(instances))
	n := 0

	for instance := range instances {
		sortedInstances[n] = instance
		n++
	}

	sort.Strings(sortedInstances)

	disks := make(map[string]struct{})
	interfaces := make(map[string]struct{})

	for _, instance := range sortedInstances {
		m := diskReadData[instance]

		for disk := range m {
			disks[disk] = struct{}{}
		}

		m = diskWriteData[instance]

		for disk := range m {
			disks[disk] = struct{}{}
		}

		m = networkReadData[instance]

		for iface := range m {
			interfaces[iface] = struct{}{}
		}

		m = networkWriteData[instance]

		for iface := range m {
			interfaces[iface] = struct{}{}
		}
	}

	sortedDisks := make([]string, len(disks))
	sortedInterfaces := make([]string, len(interfaces))

	n = 0

	for disk := range disks {
		sortedDisks[n] = disk
		n++
	}

	n = 0

	for iface := range interfaces {
		sortedInterfaces[n] = iface
		n++
	}

	// output all the data
	// 1st row - headers
	// Weird spec for '%' is to make go vet happy
	fmt.Printf("Instance,CPU Utilization (%s),Active Memory (GB),Disk Read (MB/s)\n", "%")
	for range sortedDisks {
		fmt.Print(",")
	}
	fmt.Print("Disk Write (MB/s)")
	for range sortedDisks {
		fmt.Print(",")
	}
	fmt.Print("Network Rx (MB/s)")
	for range sortedInterfaces {
		fmt.Print(",")
	}
	fmt.Print("Network Tx (MB/s)")
	for range sortedInterfaces {
		fmt.Print(",")
	}
	fmt.Println()

	// 2nd row - disks and interface names
	// printed twice for read + write
	fmt.Print(",,,")
	for _, disk := range sortedDisks {
		fmt.Print(disk)
		fmt.Print(",")
	}
	for _, disk := range sortedDisks {
		fmt.Print(disk)
		fmt.Print(",")
	}
	for _, iface := range sortedInterfaces {
		fmt.Print(iface)
		fmt.Print(",")
	}
	for _, iface := range sortedInterfaces {
		fmt.Print(iface)
		fmt.Print(",")
	}
	fmt.Println()

	// one row for each instance (hostname)
	for _, instance := range sortedInstances {
		fmt.Print(instance)
		fmt.Print(",")
		fmt.Print(cpuData.fmt(instance))
		fmt.Print(",")
		fmt.Print(memoryData.fmt(instance))
		fmt.Print(",")

		for _, disk := range sortedDisks {
			fmt.Print(diskReadData.fmt(instance, disk))
			fmt.Print(",")
		}
		for _, disk := range sortedDisks {
			fmt.Print(diskWriteData.fmt(instance, disk))
			fmt.Print(",")
		}

		for _, iface := range sortedInterfaces {
			fmt.Print(networkReadData.fmt(instance, iface))
			fmt.Print(",")
		}
		for _, iface := range sortedInterfaces {
			fmt.Print(networkWriteData.fmt(instance, iface))
			fmt.Print(",")
		}

		fmt.Println()
	}
}

func outputSingleValues(q *prom.API, query string, time time.Time) {
	// Background context => no timeout
	// queries return model.Value which needs to be cast to String, Scalar, Vector or Matrix based on v.Type()
	v, _, err := (*q).Query(context.Background(), query, time)

	if err != nil {
		logger.Println("could not execute query", query, err)
	} else {
		// Vector is an array of Samples
		for _, sample := range (v).(model.Vector) {
			// Sample.Metric is a map[string]string; Sample.Value is a float64
			if device, exists := sample.Metric["device"]; exists {
				fmt.Println(sample.Metric["instance"], device, strconv.FormatFloat(float64(sample.Value), 'f', 3, 64))
			} else {
				fmt.Println(sample.Metric["instance"], strconv.FormatFloat(float64(sample.Value), 'f', 3, 64))
			}
		}
	}
}

func collectForSingleValue(q *prom.API, query string, r *prom.Range, instances map[string]struct{}) SingleMap {
	// Background context => no timeout
	// queries return model.Value which needs to be cast to String, Scalar, Vector or Matrix based on v.Type()
	v, _, err := (*q).QueryRange(context.Background(), query, *r)

	if err != nil {
		logger.Fatalln("could not execute query", query, err)
	}

	if debug {
		logger.Println("retrieved", len(v.(model.Matrix)), "metrics for '", query, "'")
	}

	// map the instance to the average value
	values := make(map[string]float64)

	// Matrix is an array of SampleStreams
	for _, sampleStream := range v.(model.Matrix) {
		// SampleStream.Metric is a map[string]string
		// get the instance and add it to the master list, if necessary
		instance := string(sampleStream.Metric["instance"])

		if _, exists := instances[instance]; !exists {
			instances[instance] = struct{}{}

			if debug {
				logger.Println("added instance", instance)
			}
		}

		// SampleStream.Values is an array of SamplePairs
		// SamplePair contains a Time and a float64 Value
		// calculate the average
		if debug {
			logger.Println("retrieved", len(sampleStream.Values), "values for", instance)
		}

		sum := 0.0

		for _, samplePair := range sampleStream.Values {
			sum += float64(samplePair.Value)
		}

		values[instance] = sum / float64(len(sampleStream.Values))

		if debug {
			logger.Println("added", values[instance], "for", instance)
		}
	}

	return values
}

func collectForTwoValues(q *prom.API, query string, r *prom.Range, instances map[string]struct{}) DoubleMap {
	// Background context => no timeout
	// queries return model.Value which needs to be cast to String, Scalar, Vector or Matrix based on v.Type()
	v, _, err := (*q).QueryRange(context.Background(), query, *r)

	if err != nil {
		logger.Fatalln("could not execute query", query, err)
	}

	if debug {
		logger.Println("retrieved", len(v.(model.Matrix)), "metrics for '", query, "'")
	}

	// map the instance and device to the average value
	values := make(map[string]map[string]float64)

	// Matrix is an array of SampleStreams
	for _, sampleStream := range v.(model.Matrix) {
		// SampleStream.Metric is a map[string]string
		// get the instance and add it to the master list, if necessary
		instance := string(sampleStream.Metric["instance"])

		if _, exists := instances[instance]; !exists {
			instances[instance] = struct{}{}

			if debug {
				logger.Println("added instance", instance)
			}
		}

		if _, exists := values[instance]; !exists {
			values[instance] = make(map[string]float64)
		}

		device := string(sampleStream.Metric["device"])

		// SampleStream.Values is an array of SamplePairs
		// SamplePair contains a Time and a float64 Value
		// calculate the average
		if debug {
			logger.Println("retrieved", len(sampleStream.Values), "values for", instance, ":", device)
		}

		sum := 0.0

		for _, samplePair := range sampleStream.Values {
			sum += float64(samplePair.Value)
		}

		values[instance][device] = sum / float64(len(sampleStream.Values))

		if debug {
			logger.Println("added", values[instance][device], "for", instance, ":", device)
		}
	}

	return values
}

func (values SingleMap) fmt(key string) string {
	if value, exists := values[key]; exists {
		return strconv.FormatFloat(value, 'f', 3, 64)
	}
	return ""
}

func (values DoubleMap) fmt(key1 string, key2 string) string {
	if value, exists := values[key1]; exists {
		if value2, exists := value[key2]; exists {
			return strconv.FormatFloat(value2, 'f', 3, 64)
		}
		return ""
	}
	return ""
}

func parseTime(base time.Time, toParse string) (time.Time, error) {
	// try as full datetime
	toReturn, err := time.Parse(time.RFC3339, toParse)

	if err != nil {
		// try as HH:MM
		toReturn, err = time.Parse("15:04", toParse)

		if err != nil {
			// just return the base time; assume errors are checked
			return base, err
		}

		// go parsing is braindead; the HH:MM time parsed is based off the _zero_ time
		// get the value for that and the start time for today and fix it up
		zero, _ := time.Parse("15:04", "00:00")
		today := base.Truncate(time.Second * 86400)

		toReturn = time.Unix(today.Unix()+toReturn.Unix()-zero.Unix(), 0)
	}

	if debug {
		logger.Println("parsed", toParse, "to", toReturn)
	}

	return toReturn, nil
}
