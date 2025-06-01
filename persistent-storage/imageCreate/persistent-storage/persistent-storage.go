/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2018, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	metricsservice "github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix"
)

var (
	err error

	metrics       bool
	verbose       bool
	readWriteMode string
	directory     string
	blockSize     string
	fileSize      string
	numjobs       int
	jobFile       string
	testName      string
	podcount      int
	ioPing        bool

	mylog = log.New(os.Stderr, "persistent-storage: ", log.LstdFlags|log.Lshortfile)

	bm = []metricsservice.BluemixMetric{}
)

// Root Type to allow parsing fio json output
type Root struct {
	Text []*Object `json:"jobs"`
}

// Readobj type : the read part of the json
type Readobj struct {
	Bw   int     `json:"bw"`
	Iops float64 `json:"iops"`
}

// Writeobj type: the write part of the json
type Writeobj struct {
	Bw   int     `json:"bw"`
	Iops float64 `json:"iops"`
}

// Object type: the json structure from each job run
type Object struct {
	Jobname string   `json:"jobname"`
	Elapsed int      `json:"elapsed"`
	Read    Readobj  `json:"read"`
	Write   Writeobj `json:"write"`
}

func runTest(testCmd string) string {
	// #nosec G204
	cmdout, err := exec.Command("/bin/bash", "-c", testCmd).Output()
	if err != nil {
		mylog.Fatalf("Error running fio: %s\n", err)
	}
	mylog.Println(string(cmdout))
	return string(cmdout)
}

func main() {
	log.SetOutput(os.Stdout)

	flag.StringVar(&testName, "testname", "", "Test name in runAuto i.e lower case with no spaces")
	flag.StringVar(&directory, "dir", "", "Directory under persistent storage mount")
	flag.StringVar(&blockSize, "blockSize", "4k", "fio block size")
	flag.StringVar(&fileSize, "fileSize", "2G", "fio file size")
	flag.StringVar(&readWriteMode, "readWriteMode", "randread", "fio test read write mode")
	flag.StringVar(&jobFile, "jobfile", "", "full path to fio job file")
	flag.IntVar(&numjobs, "numjobs", 1, "The number of concurrent fio jobs to run")
	flag.BoolVar(&metrics, "metrics", false, "send results to IBM Metrics service")
	flag.BoolVar(&verbose, "verbose", false, "verbose output")
	flag.IntVar(&podcount, "podcount", 1, "The number of pods in the parallel group this run is part of")
	flag.BoolVar(&ioPing, "ioping", true, "Include ioping test as part of this run")

	flag.Parse()

	if len(directory) == 0 {
		mylog.Fatalf("--dir : Directory not specified")
	}

	// Create directory if it doesn't exist
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		err = os.MkdirAll(directory, 0750)
		if err != nil {
			mylog.Fatalf("Error creating directory: %s\n", err)
		}
	}

	dirElements := strings.Split(directory, string(filepath.Separator))
	individualTestName := "fio-" + dirElements[len(dirElements)-1]
	var testCmd string
	if len(jobFile) > 0 {
		testCmd = fmt.Sprintf("cd %s; fio %s --output-format=json --numjobs %d", directory, jobFile, numjobs)

		// Wait for up to 5 mins for fio config file to be available
		for i := 0; i < 20; i++ {
			if _, err := os.Stat(jobFile); os.IsNotExist(err) {
				mylog.Printf("Waiting for job file : %s\n", jobFile)
				time.Sleep(15 * time.Second)
			} else {
				break
			}
		}
	} else {
		// Use user supplied parameters or defaults. The non-supplied parameters are currently set to match
		// the default values found in the fiojobfile included as part of our parent docker image.
		jobName := fmt.Sprintf("job_%s_%s_bs", readWriteMode, blockSize)
		testCmd = fmt.Sprintf("fio --name=%s --output-format=json --randrepeat=1 --ioengine=libaio --iodepth=64 --direct=1 --gtod_reduce=1 --time_based --group_reporting --runtime=180 --ramp_time=10 --directory=%s --bs=%s --size=%s --rw=%s --numjobs=%d", jobName, directory, blockSize, fileSize, readWriteMode, numjobs)
	}
	output := runTest(testCmd)

	if metrics {
		var j Root
		err := json.Unmarshal([]byte(output), &j)
		if err != nil {
			log.Fatalf("error parsing JSON: %s\n", err.Error())
		}

		for _, t := range j.Text {
			jobName := t.Jobname
			if numjobs > 1 {
				jobName = jobName + "-" + strconv.Itoa(numjobs) + "jobs"
			}
			if podcount > 1 {
				jobName = jobName + "-" + strconv.Itoa(podcount) + "pods"
			}
			fmt.Printf("Jobname: %+v\n", t.Jobname)
			fmt.Printf("Read BW: %+v\n", t.Read.Bw)
			fmt.Printf("Write BW: %+v\n", t.Write.Bw)
			fmt.Printf("Read IOPS: %+v\n", t.Read.Iops)
			fmt.Printf("Write IOPS: %+v\n", t.Write.Iops)
			totalIOPS := t.Read.Iops + t.Write.Iops
			totalBW := t.Read.Bw + t.Write.Bw

			metricName := strings.Join([]string{"persistent_storage", individualTestName, jobName, "iops", "sparse-avg"}, ".")
			bm = append(bm, metricsservice.BluemixMetric{Name: metricName, Timestamp: time.Now().Unix(), Value: totalIOPS})

			metricName = strings.Join([]string{"persistent_storage", individualTestName, jobName, "bw", "sparse-avg"}, ".")
			bm = append(bm, metricsservice.BluemixMetric{Name: metricName, Timestamp: time.Now().Unix(), Value: totalBW})
		}
	}

	// Only run the ioping tests if requested. This allows the
	// parallel tests to skip them in the interests of time saving.
	if ioPing {
		readTestName := "ioping-" + dirElements[len(dirElements)-1]
		readTestCmd := fmt.Sprintf("ioping -c 100 %s", directory)
		writeTestName := "iopingwrite-" + dirElements[len(dirElements)-1]
		writeTestCmd := fmt.Sprintf("ioping -W -c 100 %s", directory)

		var iopingTests = map[string]string{readTestName: readTestCmd, writeTestName: writeTestCmd}

		for individualTestName, testCmd := range iopingTests {
			output = runTest(testCmd)

			if metrics {
				metricMinName := strings.Join([]string{"persistent_storage", individualTestName, "latency", "min"}, ".")
				metricAvgName := strings.Join([]string{"persistent_storage", individualTestName, "latency", "sparse-avg"}, ".")
				metricMaxName := strings.Join([]string{"persistent_storage", individualTestName, "latency", "max"}, ".")

				res := regexp.MustCompile(`min\/avg\/max\/mdev = (\d+(?:\.\d+)? [num]?s) \/ (\d+(?:\.\d+)? [num]?s) \/ (\d+(?:\.\d+)? [num]?s) \/ (\d+(?:\.\d+)? [num]?s)`).FindStringSubmatch(output)[1:]
				results := make([]time.Duration, len(res))
				for i, r := range res {
					if results[i], err = time.ParseDuration(strings.Replace(r, " ", "", -1)); err != nil {
						mylog.Fatalf("Error parsing ioping latency times: %s\n", err)
					}
				}

				bm = append(bm, metricsservice.BluemixMetric{Name: metricMinName, Timestamp: time.Now().Unix(), Value: results[0].Seconds() * 1e3})
				bm = append(bm, metricsservice.BluemixMetric{Name: metricAvgName, Timestamp: time.Now().Unix(), Value: results[1].Seconds() * 1e3})
				bm = append(bm, metricsservice.BluemixMetric{Name: metricMaxName, Timestamp: time.Now().Unix(), Value: results[2].Seconds() * 1e3})
			}
		}
	}

	if metrics {
		if verbose {
			fmt.Println(bm)
		}

		metricsservice.WriteBluemixMetrics(bm, true, testName, "")

		// As it seems to take longer to run through all of the test pods in the
		// parallel tests we scale the wait time based on the total number of pods.
		if podcount > 1 {
			log.Printf("Sleeping for %v minutes to allow runAuto.sh to collect the metrics files.\n", podcount)
			time.Sleep(time.Duration(podcount) * time.Minute)
		} else {
			log.Printf("Sleeping for 2 minutes to allow runAuto.sh to collect the metrics files.\n")
			time.Sleep(120 * time.Second)
		}

		log.Printf("Finished.\n")
	}
}
