/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2021 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	v3 "go.etcd.io/etcd/client/v3"
)

var patternCmd = &cobra.Command{
	Use:   "pattern",
	Short: "Generate key/values based on user provided pattern",
	Long:  "Generate etcd key/value pairs based on user provided pattern",

	Run: patternCmdFunc,
}

type statsBundle struct {
	statKeys  []string
	stats     map[string]string
	lastTime  time.Time
	lastStats map[string]float64
}

var (
	patTotal    int
	patPattern  string
	patPutRate  int
	patSkipInit bool

	patValChurn   int
	patLevelChurn int
	patLevel      int
	patLevelPct   int
	patTestEndKey string
	patValSpec    string

	getSerializable    bool
	getKeysOnly        bool
	fullGetRead        bool
	getRate            int
	getLevel           int
	etcdReconnectCount int
	patDoNotExit       bool

	keySpace int

	patStatsInterval       int
	patWatchCountsPerLevel string

	patFlags *flag.FlagSet

	hostIP string
)

func init() {
	RootCmd.AddCommand(patternCmd)
	patternCmd.Flags().StringVar(&patPattern, "pattern", "", "Pattern for keys/value pairs (required)")
	patternCmd.Flags().IntVar(&patTotal, "total", 1, "Total number of key/value pairs to generate (set to 0 for when churning)")
	patternCmd.Flags().IntVar(&patValChurn, "churn-val-rate", -1, "Rate limit for updating values (puts/hour, 0 for unlimited)")
	patternCmd.Flags().IntVar(&patLevelChurn, "churn-level-rate", -1, "Rate limit for deleting and putting keys at a level (updates/hour, 0 for unlimited)")
	patternCmd.Flags().IntVar(&getRate, "get-rate", -1, "Rate limit for gets (gets/hour, 0 for unlimited)")
	patternCmd.Flags().IntVar(&getLevel, "get-level", 1, "Key level to churn (0 based)")
	patternCmd.Flags().IntVar(&patLevel, "churn-level", 1, "Key level to churn (0 based)")
	patternCmd.Flags().IntVar(&patLevelPct, "churn-level-pct", 10, "% of specified level to churn")
	patternCmd.Flags().BoolVar(&patSkipInit, "skip-init", false, "Skip the loading of the initial keys and starts churn")
	patternCmd.Flags().StringVar(&patTestEndKey, "test-end-key", "/prefix/testEnd", "A key that will be watched, and when set to true the watchers will terminate")
	patternCmd.Flags().StringVar(&patValSpec, "val-spec", "0,0", "When set overrides the value regex. Comprabable to '[0-9]{n,m}'")
	patternCmd.Flags().BoolVar(&getSerializable, "serializable-gets", false, "Pass the WithSerializable option to the gets")
	patternCmd.Flags().BoolVar(&getKeysOnly, "get-keys-only", false, "Pass the WithKeysOnly option to the gets")
	patternCmd.Flags().BoolVar(&fullGetRead, "full-get-read", true, "Read back the values when doing gets")
	patternCmd.Flags().IntVar(&patStatsInterval, "stats-interval", -1, "The interval at which churn stats will be output (in seconds)")
	patternCmd.Flags().IntVar(&patPutRate, "put-rate", 0, "Number of keys to put per hour during loading of the initial keys (0 is no limit)")
	patternCmd.Flags().IntVar(&etcdReconnectCount, "etcd-reconnect-count", 0, "The number of calls to make on each connection before reconnecting (0 is never reconnect)")
	patternCmd.Flags().BoolVar(&patDoNotExit, "do-not-exit", false, "Don't exit the program after final statistics are published")
	// Watch parameters
	patternCmd.Flags().BoolVar(&treeWatchBranches, "watch-with-prefix", false, "Whether to specify 'WithPrefix' on the watch (match exact key or also sub-keys)")
	patternCmd.Flags().StringVar(&patWatchCountsPerLevel, "watch-counts-per-level", "", "The number of watchers for each level, separated by ','. Level 0 should be the first digit, followed by level 1 etc. 'n' equates to all available keys at that level. Ex: '0,0,0,n,0,0'")
	patternCmd.Flags().DurationVar(&watchPrefixGetInterval, "watch-prefix-get-interval", watchPrefixGetInterval, "The duration between requests to get keys being watched")

	patFlags = patternCmd.Flags()
}

func setupPatternStatsKeys() ([]string, map[string]string) {
	statKeys := make([]string, 38)
	stats := make(map[string]string)

	// WARNING: Don't remove existing stats, and only add stats to the end of the list. Otherwise Excel spreadsheets that depend on the order will become worthless.

	statKeys[0] = "puts"
	statKeys[1] = "puts/sec"
	statKeys[2] = "deletes"
	statKeys[3] = "deletes/sec"
	statKeys[4] = "client deletes"
	statKeys[5] = "client deletes/sec"
	statKeys[6] = "gets"
	statKeys[7] = "gets/sec"
	statKeys[8] = "client gets"
	statKeys[9] = "client gets/sec"
	statKeys[10] = "key space"
	statKeys[11] = "errors"
	statKeys[12] = "Puts mean RT(μs)"
	statKeys[13] = "Puts min RT(μs)"
	statKeys[14] = "Puts max RT(μs)"
	statKeys[15] = "Dels mean RT(μs)"
	statKeys[16] = "Dels min RT(μs)"
	statKeys[17] = "Dels max RT(μs)"
	statKeys[18] = "Gets mean RT(μs)"
	statKeys[19] = "Gets min RT(μs)"
	statKeys[20] = "Gets max RT(μs)"
	statKeys[21] = "Reconnects"
	statKeys[22] = "Reconnect mean RT(μs)"
	statKeys[23] = "Reconnect min RT(μs)"
	statKeys[24] = "Reconnect max RT(μs)"
	statKeys[25] = "puts bytes/sec"
	statKeys[26] = "gets bytes/sec"
	statKeys[27] = "watchers"
	statKeys[28] = "watch events"
	statKeys[29] = "watch events/sec"
	statKeys[30] = "prefix gets"
	statKeys[31] = "prefix gets/sec"
	statKeys[32] = "client prefix gets"
	statKeys[33] = "client prefix gets/sec"
	statKeys[34] = "Prefix gets mean RT(μs)"
	statKeys[35] = "Prefix gets min RT(μs)"
	statKeys[36] = "Prefix gets max RT(μs)"
	statKeys[37] = "prefix gets bytes/sec"

	return statKeys, stats
}

func patternCmdFunc(cmd *cobra.Command, args []string) {

	setupProfiling()
	setupCsvFile()

	if len(patPattern) == 0 {
		log.Fatal("Error: pattern is empty")
	}

	if patLevelChurn >= 0 || patValChurn >= 0 || getRate >= 0 || patWatchCountsPerLevel != "" {
		patTotal = 0
	}

	clients := make([]*v3.Client, totalClients)
	if etcdReconnectCount == 0 {
		clients = mustCreateClients(totalClients, totalConns)
	}

	patterns := make([]string, 1)
	patterns[0] = patPattern

	engine := NewPatternEngine(armadaPathRules, patterns)
	if patValSpec != "0,0" {
		if value, err := strconv.Atoi(patValSpec); err == nil {
			engine.setValueSpec(value, value)
		} else {

			split := strings.Split(patValSpec, ",")
			a, _ := strconv.Atoi(split[0])
			b, _ := strconv.Atoi(split[1])
			engine.setValueSpec(a, b)
		}
	}

	statKeys, stats := setupPatternStatsKeys()

	keySpace = engine.GetKeySpace()
	if patTotal > 0 && patTotal < keySpace {
		keySpace = patTotal
	}

	if !patSkipInit {
		requests := engine.SetupRequestChannels(clients, false)
		engine.GenerateKeys(patTotal, requests, patPutRate)

		wg.Wait()
		rampStats := engine.getChurnStats()
		totalTime := time.Now().Sub(startTime)
		patPrintStats(engine, time.Now(), totalTime, stats, statKeys, rampStats, "pattern-ramp")
	}

	treeKeyLevels = engine.MaxLevels
	watchCountsLevels := setupLevelWatchCounts(engine, patWatchCountsPerLevel)

	if patLevelChurn >= 0 || patValChurn >= 0 || getRate >= 0 || watchCountsLevels > 0 {
		engine.ResetStats()
		engine.ResetIntervalStats()
		churnStartTime := time.Now()
		lastIntervalTime = churnStartTime

		// Divy out clients favoring churn and watch
		// TODO might want to add a parameters to handle this explicitly
		var getClients []*v3.Client
		var valChurnClients []*v3.Client
		var levelChurnClients []*v3.Client
		var watchClients []*v3.Client
		if len(clients) > 4 && (patLevelChurn >= 0 || watchCountsLevels > 0) {
			currentAvailable := 0
			if getRate >= 0 {
				getClients = clients[currentAvailable : currentAvailable+1]
				currentAvailable++
			}
			if patValChurn >= 0 {
				valChurnClients = clients[currentAvailable : currentAvailable+1]
				currentAvailable++
			}
			if patLevelChurn >= 0 && watchCountsLevels > 0 {
				clientSplit := len(clients[currentAvailable:]) / 2
				levelChurnClients = clients[currentAvailable : currentAvailable+1+clientSplit]
				currentAvailable += clientSplit
				watchClients = clients[currentAvailable+clientSplit:]
			} else if patLevelChurn >= 0 {
				levelChurnClients = clients[currentAvailable:]
			} else {
				watchClients = clients[currentAvailable:]
			}
		} else {
			getClients = clients
			valChurnClients = clients
			levelChurnClients = clients
			watchClients = clients
		}

		if verbose {
			fmt.Println("client counts, \n- gets:", len(getClients), "\n- valChurn", len(valChurnClients), "\n- levelChurn", len(levelChurnClients), "\n- watch", len(watchClients))
			fmt.Println("client lists, \n- all:", clients, "\n- gets:", getClients, "\n- valChurn", valChurnClients, "\n- levelChurn", levelChurnClients, "\n- watch", watchClients)
		}

		if patLevelChurn >= 0 {
			engine.ChurnLevel(patLevel, patLevelPct, patLevelChurn, levelChurnClients)
		}

		if patValChurn >= 0 {
			engine.ChurnValues(patValChurn, valChurnClients)
		}

		if getRate >= 0 {
			fmt.Println("get clients", getClients)
			engine.GetLevel(getSerializable, getKeysOnly, getLevel, getRate, getClients)
		}

		// watch
		if watchCountsLevels > 0 {
			if watchCountsLevels != engine.MaxLevels {
				log.Printf("WARNING: watchCountsPerLevel specified %v levels, but there were %v levels in the pattern. Extra levels will be ignored and missing levels will default to 0 watchers", watchCountsLevels, engine.MaxLevels)
			}

			setupWatches(engine, watchClients)
			<-watchTreeCompletedNotifier
		}

		// Print out stats at a supplied interval
		if patStatsInterval > 0 {
			statsTicker := time.NewTicker(time.Second * time.Duration(patStatsInterval))
			go func() {
				for t := range statsTicker.C {
					if !engine.churn {
						break
					}
					intervalStats := engine.getChurnIntervalStats()
					if lastIntervalTime.IsZero() {
						// First time round so lastIntervalTime hasn't been set
						lastIntervalTime = startTime
					}
					intervalTime := t.Sub(lastIntervalTime)

					patPrintStats(engine, t, intervalTime, stats, statKeys, intervalStats, "pattern-interval")
					lastIntervalTime = t
					engine.ResetIntervalStats()
				}
			}()
		}

		engine.WaitForTestEnd(patTestEndKey)

		currentTime := time.Now()
		endStats := engine.getChurnStats()
		totalTime := currentTime.Sub(churnStartTime)
		patPrintStats(engine, startTime, totalTime, stats, statKeys, endStats, "pattern-summary")

		if patDoNotExit {
			engine.WaitForTestEnd("/donotexit")
		}
	}
}

func getIP() string {
	if hostIP == "" {
		ifaces, err := net.Interfaces()
		if err != nil {
			fmt.Println("ERROR: Couldn't get interfaces")
			os.Exit(1)
		}
	foundIP:
		for _, i := range ifaces {
			addrs, err := i.Addrs()
			if err != nil {
				fmt.Println("ERROR: Couldn't get interface address")
				os.Exit(1)
			}
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				if ip.String() != "<nil>" && ip.IsGlobalUnicast() {
					hostIP = ip.String()
					if verbose {
						fmt.Println("ip:", hostIP)
					}
					break foundIP
				}
			}
		}
	}
	return hostIP
}

func patPrintStats(engine *PatternEngine, startTime time.Time, intervalTime time.Duration, stats map[string]string, statKeys []string, statsNow churnStats, statsType string) {
	//statsNow := engine.getChurnStats()
	var intervalPutBandwidth float64
	var intervalGetBandwidth float64
	var intervalPrefixGetBandwidth float64

	intervalPutCnt := statsNow.keysPut
	intervalPutThroughput := float64(intervalPutCnt) / intervalTime.Seconds()
	intervalDelCnt := statsNow.keysDel
	intervalDelThroughput := float64(intervalDelCnt) / intervalTime.Seconds()
	intervalCDelCnt := statsNow.clientDels
	intervalCDelThroughput := float64(intervalCDelCnt) / intervalTime.Seconds()
	intervalGetCnt := statsNow.keysGet
	intervalGetThroughput := float64(intervalGetCnt) / intervalTime.Seconds()
	intervalCGetCnt := statsNow.clientGets
	intervalCGetThroughput := float64(intervalCGetCnt) / intervalTime.Seconds()
	intervalPrefixGetCnt := statsNow.keysPrefixGet
	intervalPrefixGetThroughput := float64(intervalPrefixGetCnt) / intervalTime.Seconds()
	intervalCPrefixGetCnt := statsNow.clientPrefixGets
	intervalCPrefixGetThroughput := float64(intervalCPrefixGetCnt) / intervalTime.Seconds()
	intervalErrorCnt := statsNow.errors
	intervalRecCnt := statsNow.reconnects
	intervalAveGetTime := int64(0)
	if intervalGetCnt > 0 {
		intervalAveGetTime = statsNow.keysGetTotTime / int64(intervalCGetCnt)
	}
	intervalAvePrefixGetTime := int64(0)
	if intervalPrefixGetCnt > 0 {
		intervalAvePrefixGetTime = statsNow.keysPrefixGetTotTime / int64(intervalCPrefixGetCnt)
	}
	intervalAvePutTime := int64(0)
	if intervalPutCnt > 0 {
		intervalAvePutTime = statsNow.keysPutTotTime / int64(intervalPutCnt)
	}
	intervalAveDelTime := int64(0)
	if intervalDelCnt > 0 {
		intervalAveDelTime = statsNow.keysDelTotTime / int64(intervalCDelCnt)
	}
	intervalAveRecTime := int64(0)
	if intervalRecCnt > 0 {
		intervalAveRecTime = statsNow.reconnectsTotTime / int64(intervalRecCnt)
	}
	intervalPutBandwidth = float64(statsNow.bytesPut) / intervalTime.Seconds()
	intervalGetBandwidth = float64(statsNow.bytesGet) / intervalTime.Seconds()
	intervalPrefixGetBandwidth = float64(statsNow.bytesPrefixGet) / intervalTime.Seconds()

	// Watch
	intervalWatchEventCnt := statsNow.watchEventCount
	intervalWatchEventThroughput := float64(intervalWatchEventCnt) / intervalTime.Seconds()

	log.Printf("Time: %v, interval(s): %v, putCnt: %v, puts/sec: %v, delCount: %v, dels/sec: %v, clientDelCalls: %v, client dels/sec: %v, getCount: %v, gets/sec: %v, clientGetCalls: %v, client gets/sec: %v, Errors: %v, Puts mean RT(μs): %v, Puts min RT(μs): %v, Puts max R(μs): %v, Dels mean RT(μs): %v, Dels min RT(μs): %v, Dels max R(μs): %v, Gets mean RT(μs): %v, Gets min RT(μs): %v, Gets max R(μs): %v, Reconnects: %v, Reconnect mean RT(μs): %v, Reconnect min RT(μs): %v, Reconnect max R(μs): %v, put bytes/sec: %.2f, get bytes/sec: %.2f, watchers: %v, watch events: %v, watch events/sec: %v, prefixGetCount: %v, prefixGets/sec: %v, clientPrefixGetCalls: %v, client prefixGets/sec: %v, Prefix gets mean RT(μs): %v, Prefix gets min RT(μs): %v, Prefix gets max R(μs): %v prefix get bytes/sec: %.2f",
		startTime.Format(time.StampMilli), int64(intervalTime.Seconds()),
		intervalPutCnt, strconv.FormatFloat(intervalPutThroughput, 'f', 2, 64),
		intervalDelCnt, strconv.FormatFloat(intervalDelThroughput, 'f', 2, 64),
		intervalCDelCnt, strconv.FormatFloat(intervalCDelThroughput, 'f', 2, 64),
		intervalGetCnt, strconv.FormatFloat(intervalGetThroughput, 'f', 2, 64),
		intervalCGetCnt, strconv.FormatFloat(intervalCGetThroughput, 'f', 2, 64),
		intervalErrorCnt,
		intervalAvePutTime, statsNow.keysPutMinTime, statsNow.keysPutMaxTime,
		intervalAveDelTime, statsNow.keysDelMinTime, statsNow.keysDelMaxTime,
		intervalAveGetTime, statsNow.keysGetMinTime, statsNow.keysGetMaxTime,
		intervalRecCnt, intervalAveRecTime, statsNow.reconnectsMinTime, statsNow.reconnectsMaxTime, intervalPutBandwidth, intervalGetBandwidth,
		treeWatchers, intervalWatchEventCnt, strconv.FormatFloat(intervalWatchEventThroughput, 'f', 2, 64),
		intervalPrefixGetCnt, strconv.FormatFloat(intervalPrefixGetThroughput, 'f', 2, 64),
		intervalCPrefixGetCnt, strconv.FormatFloat(intervalCPrefixGetThroughput, 'f', 2, 64),
		intervalAvePrefixGetTime, statsNow.keysPrefixGetMinTime, statsNow.keysPrefixGetMaxTime, intervalPrefixGetBandwidth)

	if len(csvFile) > 0 {
		stats[statKeys[0]] = strconv.FormatUint(uint64(intervalPutCnt), 10)
		stats[statKeys[1]] = strconv.FormatFloat(intervalPutThroughput, 'f', 4, 64)
		stats[statKeys[2]] = strconv.FormatUint(uint64(intervalDelCnt), 10)
		stats[statKeys[3]] = strconv.FormatFloat(intervalDelThroughput, 'f', 4, 64)
		stats[statKeys[4]] = strconv.FormatUint(uint64(intervalCDelCnt), 10)
		stats[statKeys[5]] = strconv.FormatFloat(intervalCDelThroughput, 'f', 4, 64)
		stats[statKeys[6]] = strconv.FormatUint(uint64(intervalGetCnt), 10)
		stats[statKeys[7]] = strconv.FormatFloat(intervalGetThroughput, 'f', 4, 64)
		stats[statKeys[8]] = strconv.FormatUint(uint64(intervalCGetCnt), 10)
		stats[statKeys[9]] = strconv.FormatFloat(intervalCGetThroughput, 'f', 4, 64)
		stats[statKeys[10]] = strconv.FormatUint(uint64(keySpace), 10)
		stats[statKeys[11]] = strconv.FormatUint(uint64(intervalErrorCnt), 10)
		stats[statKeys[12]] = strconv.FormatUint(uint64(intervalAvePutTime), 10)
		stats[statKeys[13]] = strconv.FormatUint(uint64(statsNow.keysPutMinTime), 10)
		stats[statKeys[14]] = strconv.FormatUint(uint64(statsNow.keysPutMaxTime), 10)
		stats[statKeys[15]] = strconv.FormatUint(uint64(intervalAveDelTime), 10)
		stats[statKeys[16]] = strconv.FormatUint(uint64(statsNow.keysDelMinTime), 10)
		stats[statKeys[17]] = strconv.FormatUint(uint64(statsNow.keysDelMaxTime), 10)
		stats[statKeys[18]] = strconv.FormatUint(uint64(intervalAveGetTime), 10)
		stats[statKeys[19]] = strconv.FormatUint(uint64(statsNow.keysGetMinTime), 10)
		stats[statKeys[20]] = strconv.FormatUint(uint64(statsNow.keysGetMaxTime), 10)
		stats[statKeys[21]] = strconv.FormatUint(uint64(intervalRecCnt), 10)
		stats[statKeys[22]] = strconv.FormatUint(uint64(intervalAveRecTime), 10)
		stats[statKeys[23]] = strconv.FormatUint(uint64(statsNow.reconnectsMinTime), 10)
		stats[statKeys[24]] = strconv.FormatUint(uint64(statsNow.reconnectsMaxTime), 10)
		stats[statKeys[25]] = strconv.FormatFloat(intervalPutBandwidth, 'f', 2, 64)
		stats[statKeys[26]] = strconv.FormatFloat(intervalGetBandwidth, 'f', 2, 64)
		stats[statKeys[27]] = strconv.FormatUint(uint64(treeWatchers), 10)
		stats[statKeys[28]] = strconv.FormatUint(uint64(statsNow.watchEventCount), 10)
		stats[statKeys[29]] = strconv.FormatFloat(intervalWatchEventThroughput, 'f', 4, 64)
		stats[statKeys[30]] = strconv.FormatUint(uint64(intervalPrefixGetCnt), 10)
		stats[statKeys[31]] = strconv.FormatFloat(intervalPrefixGetThroughput, 'f', 4, 64)
		stats[statKeys[32]] = strconv.FormatUint(uint64(intervalCPrefixGetCnt), 10)
		stats[statKeys[33]] = strconv.FormatFloat(intervalCPrefixGetThroughput, 'f', 4, 64)
		stats[statKeys[34]] = strconv.FormatUint(uint64(intervalAvePrefixGetTime), 10)
		stats[statKeys[35]] = strconv.FormatUint(uint64(statsNow.keysPrefixGetMinTime), 10)
		stats[statKeys[36]] = strconv.FormatUint(uint64(statsNow.keysPrefixGetMaxTime), 10)
		stats[statKeys[37]] = strconv.FormatFloat(intervalPrefixGetBandwidth, 'f', 2, 64)

		writeFile(patFlags, statsType, stats, statKeys, startTime, intervalTime)
	}
}
