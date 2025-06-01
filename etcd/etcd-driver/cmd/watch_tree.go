/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2017, 2021 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

// Copyright 2015 The etcd Authors
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
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	v3 "go.etcd.io/etcd/client/v3"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"golang.org/x/net/context"
)

// watchTreeCmd represents the watch tree command
var watchTreeCmd = &cobra.Command{
	Use:   "watch-tree",
	Short: "Benchmark watch tree",
	Long: `Benchmark watch tree simulates testing the performance of processing
watch requests and sending events to watchers to multiple etcd instances
(i.e. a tree). `,
	Run: watchTreeFunc,
}

var (
	treeWatchers int

	treeSeqKeys       bool
	treeWatchBranches bool
	treeCsvOutput     bool

	treeKeyLevels           int
	treeWatchCountsPerLevel string
	treeTestEndKey          string
	treeWatchPattern        string

	watchStatsInterval int
	lastIntervalTime   time.Time
	lastIntervalCount  int64

	levelWatchCounts []int

	nrWatchTreeCompleted int32

	treeStartTime  time.Time
	firstWatchTime time.Time
	lastWatchTime  time.Time

	watchTreeCompletedNotifier chan struct{}
	recvTreeCompletedNotifier  chan struct{}
	treeFlags                  *flag.FlagSet

	watchPrefixGetInterval time.Duration
	watchDoNotExit         bool

	getRequests     chan v3.Op
	watcherStreams  []v3.Watcher
	watchStatsMutex sync.Mutex
)

func init() {
	RootCmd.AddCommand(watchTreeCmd)

	watchTreeCmd.Flags().BoolVar(&treeSeqKeys, "sequential-keys", false, "Use sequential keys")
	watchTreeCmd.Flags().BoolVar(&treeWatchBranches, "watch-with-prefix", false, "Whether to specify 'WithPrefix' on the watch (match exact key or also sub-keys)")

	watchTreeCmd.Flags().StringVar(&treeWatchCountsPerLevel, "watch-counts-per-level", "1,5,10", "The number of watchers for each level, separated by ','. Level 0 should be the first digit, followed by level 1 etc. 'n' equates to all available keys at that level")
	watchTreeCmd.Flags().StringVar(&treeTestEndKey, "test-end-key", "/prefix/testEnd", "A key that will be watched, and when set to true the watchers will terminate")
	watchTreeCmd.Flags().StringVar(&treeWatchPattern, "pattern", "", "Pattern for keys/value pairs (required)")

	watchTreeCmd.Flags().IntVar(&watchStatsInterval, "stats-interval", -1, "The interval at which stats will be displayed (in seconds)")
	watchTreeCmd.Flags().DurationVar(&watchPrefixGetInterval, "watch-prefix-get-interval", watchPrefixGetInterval, "The duration between requests to get keys being watched")
	watchTreeCmd.Flags().BoolVar(&watchDoNotExit, "do-not-exit", false, "Don't exit the program after final statistics are published")

	treeFlags = watchTreeCmd.Flags()
}

func watchTreeFunc(cmd *cobra.Command, args []string) {

	setupProfiling()
	setupCsvFile()

	// stats
	statKeys := make([]string, 3)
	stats := make(map[string]string)

	statKeys[0] = "total watches"
	statKeys[1] = "total watch events"
	statKeys[2] = "watch events/sec"
	stats[statKeys[0]] = strconv.Itoa(treeWatchers)

	// pattern
	patterns := make([]string, 1)
	patterns[0] = treeWatchPattern
	thePatternEngine := NewPatternEngine(armadaPathRules, patterns)

	// watches
	treeKeyLevels = thePatternEngine.MaxLevels
	log.Printf("Counted %v levels in pattern %v", treeKeyLevels, treeWatchPattern)

	watchCountsLevels := setupLevelWatchCounts(thePatternEngine, treeWatchCountsPerLevel)
	if watchCountsLevels != treeKeyLevels {
		log.Printf("WARNING: watchCountsPerLevel specified %v levels, but there were %v levels in the pattern. Extra levels will be ignored and missing levels will default to 0 watchers", watchCountsLevels, treeKeyLevels)
	}

	clients := mustCreateClients(totalClients, totalConns)

	setupWatches(thePatternEngine, clients)
	<-watchTreeCompletedNotifier

	if watchStatsInterval > 0 {
		// Print out stats at a supplied interval
		treeStartTime = time.Now()
		statsTicker := time.NewTicker(time.Second * time.Duration(watchStatsInterval))
		go func() {
			for t := range statsTicker.C {
				if !thePatternEngine.churn {
					break
				}
				intervalTime, intervalCount, intervalThroughput := printStats(thePatternEngine, t)
				if len(csvFile) > 0 {
					stats[statKeys[1]] = strconv.Itoa(int(intervalCount))
					stats[statKeys[2]] = fmt.Sprintf("%4.4f", intervalThroughput)

					writeFile(treeFlags, "watch-tree-interval", stats, statKeys, t, intervalTime)

				}
			}
		}()

	}

	// Wait for test end signal to be written to etcd
	// Could probably re-use existing clients, but will use our own just in case
	testEndClient := mustCreateClients(1, 1)
	rch := testEndClient[0].Watch(context.Background(), treeTestEndKey)
	log.Printf("Created test end watch for %v", treeTestEndKey)

TestEndLoop:
	for wresp := range rch {
		for _, ev := range wresp.Events {
			if bytes.Compare(ev.Kv.Value, []byte("true")) == 0 {
				log.Print("Test End detected")
				break TestEndLoop
			}
		}
	}

	thePatternEngine.StopAllActivity()

	fmt.Printf("Total Watch Events recieved: %v\n", thePatternEngine.stats.watchEventCount)
	fmt.Printf("Time of first watch Event: %v\n", firstWatchTime.Format(time.StampMilli))
	fmt.Printf("Time of last watch Event: %v\n", lastWatchTime.Format(time.StampMilli))
	watchEventsPeriod := lastWatchTime.Sub(firstWatchTime)
	watchEventsPerSec := float64(thePatternEngine.stats.watchEventCount) / watchEventsPeriod.Seconds()
	fmt.Printf("Duration of watches: %v\n", watchEventsPeriod)
	fmt.Printf("Watch Events/sec: %v\n", strconv.FormatFloat(watchEventsPerSec, 'f', 2, 64))

	duration = watchEventsPeriod
	startTime = firstWatchTime
	stats[statKeys[1]] = strconv.Itoa(int(thePatternEngine.stats.watchEventCount))
	stats[statKeys[2]] = fmt.Sprintf("%4.4f", watchEventsPerSec)

	writeSummaryToFile(treeFlags, "watch-tree", stats, statKeys)

	if watchDoNotExit {
		thePatternEngine.WaitForTestEnd("/donotexit")
	}
}

func setupLevelWatchCounts(patternEngine *PatternEngine, watchCountsPerLevel string) int {
	if watchCountsPerLevel != "" {
		// Get the number of watchers for each level
		levelWatchCounts = make([]int, treeKeyLevels)
		countsAsStrings := strings.Split(watchCountsPerLevel, ",")
		for i := 0; i < treeKeyLevels; i++ {
			if i < len(countsAsStrings) {
				if countsAsStrings[i] == "n" {
					availableKeys := patternEngine.GetKeyPrefixes(i)
					levelWatchCounts[i] = len(availableKeys)
					treeWatchers = treeWatchers + levelWatchCounts[i]
				} else {
					s, err := strconv.Atoi(countsAsStrings[i])
					if err == nil {
						levelWatchCounts[i] = s
						treeWatchers = treeWatchers + s
					} else {
						log.Printf("WARNING: watchCountsPerLevel specified a non-integer '%v' - this level will default to 0 watchers.", countsAsStrings[i])
						levelWatchCounts[i] = 0
					}
				}
			} else {
				levelWatchCounts[i] = 0
			}
		}
		return len(countsAsStrings)
	}
	return 0
}

func setupWatches(patternEngine *PatternEngine, clients []*v3.Client) {
	watchedTrees := make([]string, treeWatchers)
	log.Printf("Total number of watchers: %v", treeWatchers)

	// Iterate over the levels getting the keys to watch
	watchTreeIndex := 0
	for i := 0; i < treeKeyLevels; i++ {
		log.Printf("Getting keys for level: %v", i)
		// Use PatternEngine to get the list of keys for this pattern
		availableKeys := patternEngine.GetKeyPrefixes(i)
		numKeys := len(availableKeys)
		if verbose {
			log.Printf("Found keys for level %v: %v", i, numKeys)
		}
		if i == (treeKeyLevels - 1) {
			log.Printf("Total keys for pattern: %v", numKeys)
		}

		nextKeyIndex := 0

		if levelWatchCounts[i] == len(availableKeys) {
			for j := 0; j < len(availableKeys); j++ {
				watchedTrees[watchTreeIndex] = availableKeys[j]
				if verbose {
					log.Printf("Adding watch for: %v", watchedTrees[watchTreeIndex])
				}
				watchTreeIndex = watchTreeIndex + 1
			}
		} else {
			for j := 0; j < levelWatchCounts[i]; j++ {

				if treeSeqKeys {
					watchedTrees[watchTreeIndex] = availableKeys[nextKeyIndex]
					if nextKeyIndex == numKeys-1 {
						nextKeyIndex = 0
					} else {
						nextKeyIndex++
					}
				} else {
					randIndex := rand.Intn(numKeys)
					watchedTrees[watchTreeIndex] = availableKeys[randIndex]
				}
				if verbose {
					log.Printf("Adding watch for: %v", watchedTrees[watchTreeIndex])
				}
				watchTreeIndex = watchTreeIndex + 1
			}
		}
	}
	if verbose {
		log.Printf("WatchedTrees: %v", watchedTrees)
	}

	requests := make(chan string, len(clients))

	watcherStreams = make([]v3.Watcher, treeWatchers)
	for i := range watcherStreams {
		watcherStreams[i] = v3.NewWatcher(clients[i%len(clients)])
	}

	atomic.StoreInt32(&nrWatchTreeCompleted, int32(0))
	watchTreeCompletedNotifier = make(chan struct{})
	for i := range watcherStreams {
		go doWatchTree(patternEngine, watcherStreams[i], requests)
	}

	go func() {
		for i := 0; i < treeWatchers; i++ {
			key := watchedTrees[i]
			requests <- key
		}
		close(requests)
	}()

	//var getRequests chan v3.Op
	if watchPrefixGetInterval > 0 {
		getRequests = patternEngine.SetupRequestChannels(clients, true)
		for i := 0; i < treeWatchers; i++ {
			key := watchedTrees[i]
			stop := patternEngine.GetStopChannel()
			startDelay := rand.Int63n(int64(watchPrefixGetInterval.Seconds())) + 1
			patternEngine.GetPrefix(false, false, key, watchPrefixGetInterval, startDelay, getRequests, stop)
		}
	}
}

func tearDownWatches() {
	if watchPrefixGetInterval > 0 {
		close(getRequests)
	}
	for i := range watcherStreams {
		watcherStreams[i].Close()
	}
}

func getWatchChannel(patternEngine *PatternEngine, stream v3.Watcher, prefix string) v3.WatchChan {
	var wch v3.WatchChan

	if treeWatchBranches {
		if verbose {
			fmt.Printf("Watching WithPrefix for %v\n", prefix)
		}
		wch = stream.Watch(context.Background(), prefix, v3.WithPrefix())
	} else {
		if verbose {
			fmt.Printf("Watching WithoutPrefix for %v\n", prefix)
		}
		wch = stream.Watch(context.Background(), prefix)
	}
	if wch == nil {
		fmt.Printf("could not open watch channel for  %v\n", prefix)
	}
	return wch
}

func doWatchTree(patternEngine *PatternEngine, stream v3.Watcher, requests <-chan string) {
	for prefix := range requests {
		go recvWatchTreeChan(patternEngine, stream, prefix)
	}
	atomic.AddInt32(&nrWatchTreeCompleted, 1)

	if atomic.LoadInt32(&nrWatchTreeCompleted) == int32(treeWatchers) {
		watchTreeCompletedNotifier <- struct{}{}
	}
}

// Part that handles the watch Events
func recvWatchTreeChan(patternEngine *PatternEngine, stream v3.Watcher, prefix string) { //(byte){
	for {
		wch := getWatchChannel(patternEngine, stream, prefix)
		for r := range wch {
			if r.Err() != nil {
				if r.Canceled {
					log.Printf("ERROR: Watch canceld, will reacquire, prefix: %v, - %v", prefix, r.Err())
					break
				} else {
					log.Printf("ERROR: Watch error, prefix: %v, - %v", prefix, r.Err())
					return
				}
			}

			if firstWatchTime.IsZero() {
				firstWatchTime = time.Now()
			}
			lastWatchTime = time.Now()

			watchStatsMutex.Lock()
			patternEngine.stats.watchEventCount = patternEngine.stats.watchEventCount + 1
			patternEngine.intervalStats.watchEventCount = patternEngine.intervalStats.watchEventCount + 1
			watchStatsMutex.Unlock()

			if verbose {
				for _, e := range r.Events {
					fmt.Printf("Got Event for prefix %v : %v\n", prefix, e)
				}
			}
		}
	}
}

func printStats(patternEngine *PatternEngine, timeNow time.Time) (time.Duration, int64, float64) {
	currentCount := patternEngine.stats.watchEventCount

	if lastIntervalTime.IsZero() {
		// First time round so lastIntervalTime hasn't been set
		lastIntervalTime = treeStartTime
	}

	intervalTime := timeNow.Sub(lastIntervalTime)
	intervalCount := currentCount - lastIntervalCount
	intervalThroughput := float64(intervalCount) / intervalTime.Seconds()

	totalTimePeriod := timeNow.Sub(firstWatchTime)
	totalThroughput := float64(currentCount) / totalTimePeriod.Seconds()

	fmt.Printf("Time: %v, Total Watchers: %v, intervalTime(s): %v, intervalCount: %v, intervalThroughput (watch events/sec): %v, Total Time(s): %v, Total Count: %v, Total Throughput (watch events/sec): %v\n", timeNow.Format(time.StampMilli), treeWatchers, int64(intervalTime.Seconds()), intervalCount, strconv.FormatFloat(intervalThroughput, 'f', 2, 64), int64(totalTimePeriod.Seconds()), currentCount, strconv.FormatFloat(totalThroughput, 'f', 2, 64))
	lastIntervalCount = currentCount
	lastIntervalTime = timeNow

	return intervalTime, intervalCount, intervalThroughput
}

// Probably not needed, but keeping code in case we want to do some puts
func doPutForWatchTree(ctx context.Context, client v3.KV, requests <-chan v3.Op) {
	for op := range requests {
		_, err := client.Do(ctx, op)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to Put for watch benchmark: %v\n", err)
			os.Exit(1)
		}
	}
}
