/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2021 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

// Copyright 2014 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// the file is borrowed from github.com/rakyll/boom/boomer/print.go

package cmd

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"encoding/csv"

	flag "github.com/spf13/pflag"
)

var (
	fileExclude      map[string]bool
	startTime        time.Time
	duration         time.Duration
	repFileInitiated bool
	repFileHeader    []string
	repFileResults   []string
)

const (
	barChar = "∎"
)

type result struct {
	errStr   string
	duration time.Duration
	happened time.Time
}

type report struct {
	avgTotal float64
	fastest  float64
	slowest  float64
	average  float64
	stddev   float64
	rps      float64

	results chan result
	total   time.Duration

	errorDist map[string]int
	lats      []float64

	sps *secondPoints
}

func printReport(results chan result) (<-chan struct{}, *report) {
	r := &report{
		results:   results,
		errorDist: make(map[string]int),
		sps:       newSecondPoints(),
	}
	return wrapReport(func() {
		r.finalize()
		r.print()
	}), r
}

func printRate(results chan result) (<-chan struct{}, *report) {
	r := &report{
		results:   results,
		errorDist: make(map[string]int),
		sps:       newSecondPoints(),
	}
	return wrapReport(func() {
		r.finalize()
		fmt.Printf(" Requests/sec:\t%4.4f\n", r.rps)
	}), r
}

func wrapReport(f func()) <-chan struct{} {
	donec := make(chan struct{})
	go func() {
		defer close(donec)
		f()
	}()
	return donec
}

func (r *report) finalize() {
	st := time.Now()
	startTime = st
	for res := range r.results {
		if res.errStr != "" {
			r.errorDist[res.errStr]++
		} else {
			r.sps.Add(res.happened, res.duration)
			r.lats = append(r.lats, res.duration.Seconds())
			r.avgTotal += res.duration.Seconds()
		}
	}
	r.total = time.Since(st)
	duration = r.total

	r.rps = float64(len(r.lats)) / r.total.Seconds()
	r.average = r.avgTotal / float64(len(r.lats))
	for i := range r.lats {
		dev := r.lats[i] - r.average
		r.stddev += dev * dev
	}
	r.stddev = math.Sqrt(r.stddev / float64(len(r.lats)))
}

func (r *report) print() {
	sort.Float64s(r.lats)

	if len(r.lats) > 0 {
		r.fastest = r.lats[0]
		r.slowest = r.lats[len(r.lats)-1]
		fmt.Printf("\nSummary:\n")
		fmt.Printf("  Total:\t%4.4f secs.\n", r.total.Seconds())
		fmt.Printf("  Slowest:\t%4.4f secs.\n", r.slowest)
		fmt.Printf("  Fastest:\t%4.4f secs.\n", r.fastest)
		fmt.Printf("  Average:\t%4.4f secs.\n", r.average)
		fmt.Printf("  Stddev:\t%4.4f secs.\n", r.stddev)
		fmt.Printf("  Requests/sec:\t%4.4f\n", r.rps)
		r.printHistogram()
		r.printLatencies()
		/*		if sample {
					r.printSecondSample()
				}
		*/
	}

	if len(r.errorDist) > 0 {
		r.printErrors()
	}
}

func (r *report) extractStats() (map[string]string, []string) {
	sort.Float64s(r.lats)
	pctls := []int{95}
	var stats map[string]string
	var keys []string
	index := 0

	if len(r.lats) > 0 {
		stats = make(map[string]string)
		keys = make([]string, 5+len(pctls))
		r.fastest = r.lats[0]
		r.slowest = r.lats[len(r.lats)-1]
		stats["Slowest (secs)"] = fmt.Sprintf("%4.4f", r.slowest)
		stats["Fastest (secs)"] = fmt.Sprintf("%4.4f", r.fastest)
		stats["Average (secs)"] = fmt.Sprintf("%4.4f", r.average)
		stats["Stddev (secs)"] = fmt.Sprintf("%4.4f", r.stddev)
		stats["Requests/sec"] = fmt.Sprintf("%4.4f", r.rps)
		keys[index] = "Slowest (secs)"
		index++
		keys[index] = "Fastest (secs)"
		index++
		keys[index] = "Average (secs)"
		index++
		keys[index] = "Stddev (secs)"
		index++
		keys[index] = "Requests/sec"
		index++

		data := r.extractLatencies(pctls)
		for i := 0; i < len(pctls); i++ {
			name := fmt.Sprintf("%vth percentile", pctls[i])
			keys[index] = name
			index++
			if data[i] > 0 {
				stats[name] = fmt.Sprintf("%4.4f", data[i])
			} else {
				stats[name] = ""
			}
		}
	}

	return stats, keys
}

// Prints percentile latencies.
func (r *report) printLatencies() {
	pctls := []int{10, 25, 50, 75, 90, 95, 99}
	data := make([]float64, len(pctls))
	j := 0
	for i := 0; i < len(r.lats) && j < len(pctls); i++ {
		current := i * 100 / len(r.lats)
		if current >= pctls[j] {
			data[j] = r.lats[i]
			j++
		}
	}
	fmt.Printf("\nLatency distribution:\n")
	for i := 0; i < len(pctls); i++ {
		if data[i] > 0 {
			fmt.Printf("  %v%% in %4.4f secs.\n", pctls[i], data[i])
		}
	}
}

func (r *report) extractLatencies(pctls []int) []float64 {
	data := make([]float64, len(pctls))
	j := 0
	for i := 0; i < len(r.lats) && j < len(pctls); i++ {
		current := i * 100 / len(r.lats)
		if current >= pctls[j] {
			data[j] = r.lats[i]
			j++
		}
	}
	return data
}

func (r *report) printSecondSample() {
	fmt.Println(r.sps.getTimeSeries())
}

func (r *report) printHistogram() {
	bc := 10
	buckets := make([]float64, bc+1)
	counts := make([]int, bc+1)
	bs := (r.slowest - r.fastest) / float64(bc)
	for i := 0; i < bc; i++ {
		buckets[i] = r.fastest + bs*float64(i)
	}
	buckets[bc] = r.slowest
	var bi int
	var max int
	for i := 0; i < len(r.lats); {
		if r.lats[i] <= buckets[bi] {
			i++
			counts[bi]++
			if max < counts[bi] {
				max = counts[bi]
			}
		} else if bi < len(buckets)-1 {
			bi++
		}
	}
	fmt.Printf("\nResponse time histogram:\n")
	for i := 0; i < len(buckets); i++ {
		// Normalize bar lengths.
		var barLen int
		if max > 0 {
			barLen = counts[i] * 40 / max
		}
		fmt.Printf("  %4.3f [%v]\t|%v\n", buckets[i], counts[i], strings.Repeat(barChar, barLen))
	}
}

func (r *report) printErrors() {
	fmt.Printf("\nError distribution:\n")
	for err, num := range r.errorDist {
		fmt.Printf("  [%d]\t%s\n", num, err)
	}
}

func addToFileExclude(param string) {
	if fileExclude == nil {
		fileExclude = make(map[string]bool)
	}
	fileExclude[param] = true
}

func writeSummaryToFile(flags *flag.FlagSet, testName string, stats map[string]string, keys []string) {
	writeFile(flags, testName, stats, keys, startTime, duration)
}

func writeFile(flags *flag.FlagSet, testName string, stats map[string]string, keys []string, statTime time.Time, duration time.Duration) {

	if len(csvFile) > 0 {
		file, err := os.OpenFile(csvFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			fmt.Println("Error openning file", err)
			os.Exit(1)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		defer writer.Flush()

		var column int
		if repFileInitiated == false {
			// Only do the initialalization once for case where multiple lines are written
			columns := 0
			flags.VisitAll(func(flg *flag.Flag) {
				if fileExclude[flg.Name] == false {
					columns++
				}
			})

			columns += len(stats) + 3
			repFileHeader = make([]string, columns)
			repFileResults = make([]string, columns)
			repFileHeader[0] = "test"
			repFileHeader[1] = "startTime"
			repFileHeader[2] = "duration (ms)"

			column = 3
			for _, name := range keys {
				repFileHeader[column] = name
				column++
			}

			flags.VisitAll(func(flg *flag.Flag) {
				if fileExclude[flg.Name] == false {
					repFileHeader[column] = flg.Name
					if flg.Name == "endpoints" {
						repFileResults[column] = strconv.Itoa(len(endpoints))
					} else if flg.Name == "file-comment" {
						repFileResults[column] = flg.Value.String()
					} else {
						repFileResults[column] = flg.Value.String()
					}
					column++
				}
			})

			stats, _ := file.Stat()
			if stats.Size() == 0 {
				err = writer.Write(repFileHeader)
			}

			if err != nil {
				fmt.Println("File write error for header")
			}

			repFileInitiated = true
		}

		repFileResults[0] = testName
		repFileResults[1] = statTime.Format("2006-01-02 15:04:05")
		repFileResults[2] = strconv.FormatInt(int64(duration/time.Millisecond), 10)
		column = 3

		for _, name := range keys {
			repFileResults[column] = stats[name]
			column++
		}

		err = writer.Write(repFileResults)

		if err != nil {
			fmt.Println("File write error for results")
		}

	}
}
