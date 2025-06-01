/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

// Copyright 2016 The etcd Authors
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

package cmd

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"
)

type timeSeries struct {
	timestamp  int64
	avgLatency time.Duration
	throughPut int64
}

// TimeSeries ...
type TimeSeries []timeSeries

func (ts TimeSeries) Swap(i, j int)      { ts[i], ts[j] = ts[j], ts[i] }
func (ts TimeSeries) Len() int           { return len(ts) }
func (ts TimeSeries) Less(i, j int) bool { return ts[i].timestamp < ts[j].timestamp }

type secondPoint struct {
	totalLatency time.Duration
	count        int64
}

type secondPoints struct {
	mu sync.Mutex
	tm map[int64]secondPoint
}

func newSecondPoints() *secondPoints {
	return &secondPoints{tm: make(map[int64]secondPoint)}
}

// Add ...
func (sp *secondPoints) Add(ts time.Time, lat time.Duration) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	tk := ts.Unix()
	if v, ok := sp.tm[tk]; !ok {
		sp.tm[tk] = secondPoint{totalLatency: lat, count: 1}
	} else {
		v.totalLatency += lat
		v.count++
		sp.tm[tk] = v
	}
}

func (sp *secondPoints) getTimeSeries() TimeSeries {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	var (
		minTs int64 = math.MaxInt64
		maxTs int64 = -1
	)
	for k := range sp.tm {
		if minTs > k {
			minTs = k
		}
		if maxTs < k {
			maxTs = k
		}
	}
	for ti := minTs; ti < maxTs; ti++ {
		if _, ok := sp.tm[ti]; !ok { // fill-in empties
			sp.tm[ti] = secondPoint{totalLatency: 0, count: 0}
		}
	}

	var (
		tslice = make(TimeSeries, len(sp.tm))
		i      int
	)
	for k, v := range sp.tm {
		var lat time.Duration
		if v.count > 0 {
			lat = time.Duration(v.totalLatency) / time.Duration(v.count)
		}
		tslice[i] = timeSeries{
			timestamp:  k,
			avgLatency: lat,
			throughPut: v.count,
		}
		i++
	}

	sort.Sort(tslice)
	return tslice
}

// String ...
func (ts TimeSeries) String() string {
	buf := new(bytes.Buffer)
	wr := csv.NewWriter(buf)
	if err := wr.Write([]string{"unix_ts", "avg_latency", "throughput"}); err != nil {
		log.Fatal(err)
	}
	rows := [][]string{}
	for i := range ts {
		row := []string{
			fmt.Sprintf("%d", ts[i].timestamp),
			fmt.Sprintf("%s", ts[i].avgLatency),
			fmt.Sprintf("%d", ts[i].throughPut),
		}
		rows = append(rows, row)
	}
	if err := wr.WriteAll(rows); err != nil {
		log.Fatal(err)
	}
	wr.Flush()
	if err := wr.Error(); err != nil {
		log.Fatal(err)
	}
	return fmt.Sprintf("\nSample in one second (unix latency throughput):\n%s", buf.String())
}
