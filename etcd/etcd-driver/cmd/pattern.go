/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
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
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	regen "github.com/zach-klippenstein/goregen"
	v3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/context"
)

// Type of key maching algorithm
const (
	TypeKeyGen   = "KeyGen"
	TypePatterns = "Patterns"
	TypePattern  = "Pattern"
	TypeFormat   = "Format"
)

const digits = "0123456789"

var (
	putStatsMutex       sync.Mutex
	getStatsMutex       sync.Mutex
	delStatsMutex       sync.Mutex
	reconnectStatsMutex sync.Mutex
)

type patternRule struct {
	nodeType string
	leaf     bool
	patterns []string
	pattern  string
	max      int
	cnt      int
	valueGen regen.Generator
	keyGen   regen.Generator
	args     []string
	minValue int
	maxValue int
}

// PatternEngine ...
type PatternEngine struct {
	builderRules map[string]patternRule
	keyRules     [][]patternRule
	expectedKeys int
	MaxLevels    int
	churn        bool
	stopChannels []chan bool

	stats         churnStats
	intervalStats churnStats

	// Store the keys so they can be accessed by the watchers
	// Will support a maximum of 10 levels
	keyList [][]string
}

type churnStats struct {
	keysPut        uint32
	keysPutTotTime int64
	keysPutMinTime int64
	keysPutMaxTime int64
	bytesPut       int64

	keysDel        uint32
	clientDels     uint32
	keysDelTotTime int64
	keysDelMinTime int64
	keysDelMaxTime int64

	keysGet        uint32
	clientGets     uint32
	keysGetTotTime int64
	keysGetMinTime int64
	keysGetMaxTime int64
	bytesGet       int64

	errors uint32

	reconnects        uint32
	reconnectsTotTime int64
	reconnectsMinTime int64
	reconnectsMaxTime int64

	watchEventCount int64

	keysPrefixGet        uint32
	clientPrefixGets     uint32
	keysPrefixGetTotTime int64
	keysPrefixGetMinTime int64
	keysPrefixGetMaxTime int64
	bytesPrefixGet       int64
}

func (e *churnStats) reset() {
	e.keysDelMaxTime = 0
	e.keysGetMaxTime = 0
	e.keysPrefixGetMaxTime = 0
	e.keysPutMaxTime = 0
	e.clientDels = 0
	e.clientGets = 0
	e.clientPrefixGets = 0
	e.keysDel = 0
	e.keysDelMinTime = 0
	e.keysDelTotTime = 0
	e.keysGet = 0
	e.keysGetMinTime = 0
	e.keysGetTotTime = 0
	e.keysPrefixGet = 0
	e.keysPrefixGetMinTime = 0
	e.keysPrefixGetTotTime = 0
	e.keysPut = 0
	e.keysPutMinTime = 0
	e.keysPutTotTime = 0
	e.bytesGet = 0
	e.bytesPrefixGet = 0
	e.bytesPut = 0
	e.errors = 0
	e.reconnects = 0
	e.reconnectsMaxTime = 0
	e.reconnectsMinTime = 0
	e.reconnectsTotTime = 0
	e.watchEventCount = 0
}

type patternGenerator struct {
	expectedKeys int
	keysAdded    int
}

// NewPatternEngine Create the pattern engine
func NewPatternEngine(builderRules map[string]patternRule, patterns []string) *PatternEngine {

	engine := PatternEngine{builderRules: make(map[string]patternRule, len(builderRules)),
		keyRules:     make([][]patternRule, len(patterns)),
		expectedKeys: 0,
		churn:        true,
		keyList:      make([][]string, 10)}

	rand.Seed(time.Now().UnixNano())
	for r, rl := range builderRules {
		if rl.max > 0 && rl.cnt > rl.max {
			log.Fatalf("Value of --%s (%v) is greater than maximum allowed (%v)", r, rl.cnt, rl.max)
		}
		rl.init()
		engine.builderRules[r] = rl
	}

	for k, pat := range patterns {
		engine.keyRules[k] = engine.parsePath(pat)
	}

	engine.setMaxLevels()
	engine.ResetStats()
	return &engine
}

func (e *PatternEngine) setMaxLevels() {
	e.MaxLevels = 0
	for _, rules := range e.keyRules {
		if e.MaxLevels < len(rules) {
			e.MaxLevels = len(rules)
		}
	}
}

// GetKeySpace ..
func (e *PatternEngine) GetKeySpace() int {
	keys := 0
	for _, rules := range e.keyRules {
		ruleKeys := 1
		for _, rule := range rules {
			ruleKeys = ruleKeys * rule.cnt
		}
		keys = keys + ruleKeys
	}
	return keys
}

// ResetStats clears data from pervious operations
func (e *PatternEngine) ResetStats() {
	e.stats.reset()
	startTime = time.Now()
}

// ResetIntervalStats clears the interval stats
func (e *PatternEngine) ResetIntervalStats() {
	e.intervalStats.reset()
}

// GetKeyPrefixes returns the prefixLevel prefex for the keys
func (e *PatternEngine) GetKeyPrefixes(prefixLevel int) []string {
	if len(e.keyList[prefixLevel]) == 0 || e.keyList[prefixLevel][0] == "" {
		e.keyList[prefixLevel] = []string{""}
		gen := patternGenerator{expectedKeys: 0, keysAdded: 0}
		for _, rules := range e.keyRules {
			gen.constructKey("", rules, 0, prefixLevel, func(level int, key string, value string) {
				if level == prefixLevel {
					if e.keyList[level][0] == "" {
						e.keyList[level][0] = key
					} else {
						e.keyList[level] = append(e.keyList[level], key)
					}
				}
			})
		}
	}
	return e.keyList[prefixLevel]
}

// GenerateKeys creates the keys and puts them to etcd
func (e *PatternEngine) GenerateKeys(keys int, requests chan v3.Op, rate int) {
	go func() {
		gen := patternGenerator{expectedKeys: keys, keysAdded: 0}
		var ticker *time.Ticker
		if rate > 0 {
			ticker = time.NewTicker(time.Hour / time.Duration(rate))
			defer ticker.Stop()
		}

		log.Print("Generating and adding key/values")
		for ok := true; ok; {
			for _, rules := range e.keyRules {
				gen.constructKey("", rules, 0, -1, func(level int, key string, value string) {
					if rate > 0 {
						<-ticker.C
					}
					if verbose {
						log.Printf("Requesting put (GenerateKeys) for %v : %v", key, value)
					}
					requests <- v3.OpPut(key, value)
				})

				if verbose {
					log.Printf("Added %v key/values", gen.keysAdded)
				}
				if gen.expectedKeys > 0 && gen.keysAdded >= gen.expectedKeys {
					ok = false
					break
				}
			}
			if gen.expectedKeys == 0 {
				ok = false
			}
		}
		close(requests)
	}()
}

// ChurnValues puts new values for existing keys to create load
func (e *PatternEngine) ChurnValues(rate int, clients []*v3.Client) {
	if len(e.keyRules) > 1 {
		log.Print("Failure: ChurnValues() can't be used with more than 1 pattern")
		os.Exit(1)
	}

	keys := e.GetKeyPrefixes(e.MaxLevels - 1)
	preCnt := len(keys)

	requests := e.SetupRequestChannels(clients, false)

	var ticker *time.Ticker
	if rate > 0 {
		ticker = time.NewTicker(time.Hour / time.Duration(rate))
	}

	go func() {
		for {
			if rate > 0 {
				<-ticker.C
			}
			if !e.churn {
				break
			}
			indx := rand.Intn(preCnt)
			key := keys[indx]
			value := getValue(e.keyRules[0][e.MaxLevels-1])
			if verbose {
				log.Printf("Requesting put (ChurnValues) for %v : %v", key, value)
			}
			requests <- v3.OpPut(key, value)
		}
		close(requests)
	}()
}

// ChurnLevel puts and deletes key/values below a certain level in the tree
func (e *PatternEngine) ChurnLevel(level int, percent int, rate int, clients []*v3.Client) {
	if len(e.keyRules) > 1 {
		log.Print("Failure: ChurnLevel() can't be used with more than 1 pattern")
		os.Exit(1)
	}

	if len(e.keyRules[0])-1 <= level {
		log.Print("Failure: ChurnLevel() level requested is greater than number of levels in pattern")
		os.Exit(1)
	}

	prefixes := e.GetKeyPrefixes(level)
	preCnt := len(prefixes)
	churnCnt := preCnt * percent / 100
	churnRef := make([]int, churnCnt)

	requests := e.SetupRequestChannels(clients, true)

	gen := patternGenerator{expectedKeys: 0, keysAdded: 0}

	var ticker *time.Ticker
	if rate > 0 {
		ticker = time.NewTicker(time.Hour / time.Duration(rate))
	}

	go func() {
		delete := true
		indx := 0
		for {
			if rate > 0 {
				<-ticker.C
			}
			if !e.churn {
				break
			}
			if delete {
				// Delete all keys with given prefix
				p := -1
				for p == -1 {
					p = rand.Intn(preCnt)
					// Check that key hasn't previouly been deleted.
					for i := 0; i < indx; i++ {
						if p == churnRef[i] {
							p = -1
							break
						}
					}
				}
				prefix := prefixes[p]
				if verbose {
					log.Printf("Requesting delete prefix (ChurnLevel) for %v", prefix)
				}
				requests <- v3.OpDelete(prefix, v3.WithPrefix())

				churnRef[indx] = p
			} else {
				// Create all keys under a given prefix
				prefix := prefixes[churnRef[indx]]
				gen.constructKey(prefix, e.keyRules[0], level+1, -1, func(level int, key string, value string) {
					if verbose {
						log.Printf("Requesting put (ChurnLevel) for %v:%v", key, value)
					}
					requests <- v3.OpPut(key, value)
				})
				churnRef[indx] = -1
			}

			indx++
			if indx >= churnCnt {
				indx = 0
				if delete {
					delete = false
				} else {
					delete = true
				}
			}
		}
		close(requests)
	}()
}

// GetLevel gets all keys/values for randomly selected prefixes at a certain level
func (e *PatternEngine) GetLevel(serializable bool, keysOnly bool, level int, rate int, clients []*v3.Client) {
	if len(e.keyRules) > 1 {
		log.Print("Failure: GetLevel() can't be used with more than 1 pattern")
		os.Exit(1)
	}

	if len(e.keyRules[0])-1 < level {
		log.Print("Failure: GetLevel() level requested is greater than number of levels in pattern")
		os.Exit(1)
	}

	prefixes := e.GetKeyPrefixes(level)
	preCnt := len(prefixes)

	isLeaf := e.keyRules[0][level].leaf

	requests := e.SetupRequestChannels(clients, !isLeaf)

	var ticker *time.Ticker
	if rate > 0 {
		ticker = time.NewTicker(time.Hour / time.Duration(rate))
	}

	go func() {
		for {
			if rate > 0 {
				<-ticker.C
			}
			if !e.churn {
				break
			}
			p := rand.Intn(preCnt)
			prefix := prefixes[p]
			if verbose {
				log.Printf("Requesting Get prefix (GetLevel) for %v", prefix)
			}
			var ops []v3.OpOption
			if !isLeaf {
				ops = make([]v3.OpOption, 1)
				ops[0] = v3.WithPrefix()
			}
			if serializable {
				ops = append(ops, v3.WithSerializable())
			}
			if keysOnly {
				ops = append(ops, v3.WithKeysOnly())
			}
			requests <- v3.OpGet(prefix, ops...)
		}
		close(requests)
	}()
}

// GetPrefix gets all keys/values for requested prefix
func (e *PatternEngine) GetPrefix(serializable bool, keysOnly bool, prefix string, rate time.Duration, startDelay int64, requests chan v3.Op, stop chan bool) {

	go func() {
		startTicker := time.NewTicker(time.Second * time.Duration(startDelay))
		<-startTicker.C
		startTicker.Stop()

		var ticker *time.Ticker
		if rate > 0 {
			ticker = time.NewTicker(rate)
		} else {
			ticker = time.NewTicker(time.Millisecond)
		}
		for {
			select {
			case <-ticker.C:
				if !e.churn {
					break
				}
				if verbose {
					log.Printf("Requesting Get prefix for %v", prefix)
				}
				ops := make([]v3.OpOption, 1)
				ops[0] = v3.WithPrefix()
				if serializable {
					ops = append(ops, v3.WithSerializable())
				}
				if keysOnly {
					ops = append(ops, v3.WithKeysOnly())
				}
				requests <- v3.OpGet(prefix, ops...)
			case <-stop:
				break
			}
		}
	}()
}

// GetStopChannel retrieve a channel that will be used to stop requests to etcd
func (e *PatternEngine) GetStopChannel() chan bool {
	stopChannel := make(chan bool, 1)
	e.stopChannels = append(e.stopChannels, stopChannel)
	return stopChannel
}

// StopAllActivity sends a message to all the stop channels to shutdown all etcd requests
func (e *PatternEngine) StopAllActivity() {
	e.churn = false
	for _, stopChannel := range e.stopChannels {
		stopChannel <- true
	}
	tearDownWatches()
}

// SetupRequestChannels setup for pipeline of etcd requests
func (e *PatternEngine) SetupRequestChannels(clients []*v3.Client, prefixGets bool) chan v3.Op {

	requests := make(chan v3.Op, len(clients))

	for i := range clients {
		wg.Add(1)
		go e.doOps(clients[i], requests, i, prefixGets)
	}
	return requests
}

// WaitForTestEnd wait for the previously initiated test to complete
func (e *PatternEngine) WaitForTestEnd(testEndKey string) {
	// Wait for test end signal to be written to etcd
	// Could probably re-use existing clients, but will use our own just in case
	testEndClient := mustCreateClients(1, 1)
	log.Printf("Created test end watch for %v", testEndKey)

TestEndLoop:
	for {
		rch := testEndClient[0].Watch(context.Background(), testEndKey)
		for wresp := range rch {
			if wresp.Canceled {
				log.Printf("Test end watch canceled and will be recreated")
				break
			} else {
				for _, ev := range wresp.Events {
					if bytes.Compare(ev.Kv.Value, []byte("true")) == 0 {
						log.Printf("Test End detected")
						e.StopAllActivity()
						break TestEndLoop
					}
				}
			}
		}
	}
	log.Println("Test End loop exited")
	wg.Wait()
}

func (e *PatternEngine) setValueSpec(min int, max int) {
	generator, _ := regen.NewGenerator("[0-9]", &regen.GeneratorArgs{
		RngSource: rand.NewSource(time.Now().UnixNano()),
	})
	for _, rules := range e.keyRules {
		leaf := len(rules) - 1
		rules[leaf].minValue = min
		rules[leaf].maxValue = max
		rules[leaf].valueGen = generator
	}
}

func getValue(rule patternRule) string {
	if rule.minValue >= 0 {
		var result []byte
		cnt := rule.minValue + rand.Intn(rule.maxValue-rule.minValue)
		idx := rand.Intn(len(digits))
		for i := 0; i < cnt; i++ {
			if cnt <= 10 || i%(cnt/10) == 0 {
				idx = rand.Intn(len(digits))
			}
			result = append(result, digits[idx])
		}
		return string(result[:])
	}
	var resultBuf bytes.Buffer
	resultBuf.WriteString(rule.valueGen.Generate())
	return resultBuf.String()
}

func (r *patternRule) init() {
	if r.nodeType == TypePattern && len(r.pattern) > 0 {
		generator, err := regen.NewGenerator(r.pattern, &regen.GeneratorArgs{
			RngSource: rand.NewSource(time.Now().UnixNano()),
		})
		if err != nil {
			log.Printf("Failed: regen pattern error: %s", err)
		}
		r.patterns = make([]string, r.cnt)
		for i := 0; i < r.cnt; i++ {
			r.patterns[i] = generator.Generate()
		}
		r.nodeType = TypePatterns
	} else if len(r.patterns) < r.cnt {
		log.Printf("Failed initialization: len(paterns)=%d < r.cnt=%d, r.max=%d", len(r.patterns), r.cnt, r.max)
	}
}

func getCmdCnt(path string) (string, int) {
	var cmd string
	var cnt = 1
	split := strings.Split(path, "[")
	if len(split) > 1 {
		cmd = split[0][1:]
		cnt, _ = strconv.Atoi(split[1][:len(split[1])-1])
	} else {
		cmd = path[1:]
	}
	return cmd, cnt
}

func (e *PatternEngine) parsePath(pattern string) []patternRule {
	paths := strings.Split(pattern, "/")
	fields := make([]patternRule, 0, len(paths))
	fieldIndex := 0
	leaf := false
	for _, path := range paths {
		if len(path) == 0 {
			continue
		} else if leaf == true {
			log.Print("Failure: Leaf specified at a non leaf level")
			os.Exit(1)
		}

		var vGen regen.Generator

		if strings.Contains(path, ";") {
			// value spec: ".../...;[a-z0-9]{1000}"
			tagRegex := strings.Split(path, ";")
			path = tagRegex[0]
			var err error
			vGen, err = regen.NewGenerator(tagRegex[1], nil)
			if err != nil {
				log.Printf("Failed: regen pattern error: %s/n", err)
			}
			leaf = true
		}

		if strings.HasPrefix(path, ":") {
			// builder rules: ".../:<rule name>/..."
			cmd, cnt := getCmdCnt(path)
			if rl, ok := e.builderRules[cmd]; ok {
				rl.valueGen = vGen
				rl.leaf = leaf
				if cnt == -1 {
					rl.cnt = cnt
				}
				fields = append(fields, rl)
			} else {
				log.Printf("No match for %s", path[1:])
			}
			fieldIndex++
		} else if strings.HasPrefix(path, "!") {
			// random key generator: ".../![a-z0-9]{80}/..." -> pattern generator gets "[a-z0-9]{80}"
			gen, err := regen.NewGenerator(path[1:], nil)
			if err != nil {
				log.Printf("Failed: regen pattern error: %s", err)
			}
			fields = append(fields, patternRule{nodeType: TypeKeyGen, leaf: leaf, pattern: path[1:], cnt: 1, max: 1, keyGen: gen, valueGen: vGen, minValue: -1})
			fieldIndex++
		} else if strings.HasPrefix(path, "%") {
			// key string: " ../%tel-%04d/.." -> "tel-%04d" is fed to a Printf statement
			// key string: " ../%tel-%04d[8]/.." the "[8] sets the cnt to 8, otherwise it is 1. Values will 0-(n-1)"
			cmd, cnt := getCmdCnt(path)
			fields = append(fields, patternRule{nodeType: TypeFormat, leaf: leaf, pattern: cmd, cnt: cnt, max: 0, valueGen: vGen, minValue: -1})
			fieldIndex++
		} else {
			// fixed string: "../client/.."
			fields = append(fields, patternRule{nodeType: TypePatterns, leaf: leaf, patterns: []string{path}, cnt: 1, max: 1, valueGen: vGen, minValue: -1})
			fieldIndex++
		}
		if verbose {
			fmt.Println("Level pattern rule:", fields[fieldIndex-1])
		}
	}
	if leaf == false {
		log.Printf("Failure: Pattern doesn't have a value specification: %s", pattern)
		os.Exit(1)
	}
	return fields
}

func (g *patternGenerator) constructKey(key string, rules []patternRule, level int, maxLevel int, put func(level int, key string, value string)) {
	for i := 0; i < rules[level].cnt; i++ {

		nextKey := key + "/" + rules[level].genKeyLevel(level, i, g.keysAdded)

		if maxLevel >= 0 && maxLevel == level {
			put(level, nextKey, "")
		} else if rules[level].leaf {
			put(level, nextKey, getValue(rules[level]))
			g.keysAdded++
		} else {
			g.constructKey(nextKey, rules, level+1, maxLevel, put)
		}
		if g.expectedKeys > 0 && g.keysAdded >= g.expectedKeys {
			break
		}
	}
}

func (r *patternRule) genKeyLevel(level int, instance int, keysAdded int) string {
	levelStr := ""

	switch r.nodeType {
	case TypePatterns:
		levelStr = r.patterns[instance]
	case TypeKeyGen:
		levelStr = r.keyGen.Generate()
	case TypeFormat:
		levelStr = fmt.Sprintf(r.pattern, instance)
	default:
		log.Print("CHECK THIS OUT: I don't expect it to be used")
		log.Printf("rules[%d] = %v\n", level, r)
	}

	return levelStr
}

func (e *PatternEngine) getChurnStats() (stats churnStats) {
	return e.stats
}
func (e *PatternEngine) getChurnIntervalStats() (stats churnStats) {
	return e.intervalStats
}

func (e *PatternEngine) doOps(client v3.KV, requests <-chan v3.Op, clientNum int, prefixGets bool) {
	defer wg.Done()
	useCount := 0
	var localCon *v3.Client
	if etcdReconnectCount > 0 {
		localCon = mustCreateConn(true)
		client = localCon.KV
	}

	for op := range requests {
		if etcdReconnectCount > 0 {
			if useCount == etcdReconnectCount {
				conSt := time.Now()
				localCon.Close()
				localCon = mustCreateConn(true)
				client = localCon.KV
				useCount = 0
				// Store as Microseconds
				conRespTime := time.Since(conSt).Nanoseconds() / 1000
				if verbose {
					log.Printf("Client %v reconnected after %v uses took %v microseconds", clientNum, etcdReconnectCount, conRespTime)
				}
				reconnectStatsMutex.Lock()
				e.stats.reconnects = e.stats.reconnects + 1
				e.intervalStats.reconnects = e.intervalStats.reconnects + 1
				e.stats.reconnectsTotTime = e.stats.reconnectsTotTime + conRespTime
				e.intervalStats.reconnectsTotTime = e.intervalStats.reconnectsTotTime + conRespTime
				if e.stats.reconnectsMinTime == 0 || conRespTime < e.stats.reconnectsMinTime {
					e.stats.reconnectsMinTime = conRespTime
				}
				if e.intervalStats.reconnectsMinTime == 0 || conRespTime < e.intervalStats.reconnectsMinTime {
					e.intervalStats.reconnectsMinTime = conRespTime
				}
				if e.stats.reconnectsMaxTime == 0 || conRespTime > e.stats.reconnectsMaxTime {
					e.stats.reconnectsMaxTime = conRespTime
				}
				if e.intervalStats.reconnectsMaxTime == 0 || conRespTime > e.intervalStats.reconnectsMaxTime {
					e.intervalStats.reconnectsMaxTime = conRespTime
				}
				reconnectStatsMutex.Unlock()
			}
			useCount = useCount + 1
		}
		var ctx context.Context
		var cancel context.CancelFunc
		if clientTimeout > -1 {
			ctx, cancel = context.WithTimeout(context.Background(), time.Duration(clientTimeout)*time.Second)

		} else {
			ctx = context.Background()
		}

		st := time.Now()
		resp, err := client.Do(ctx, op)

		// Store as Microseconds
		respTime := time.Since(st).Nanoseconds() / 1000

		if clientTimeout > -1 {
			cancel()
		}

		if err != nil {
			opType := "Other"
			if op.IsPut() {
				opType = "Put"
			} else if op.IsGet() {
				opType = "Get"
			} else if op.IsDelete() {
				opType = "Delete"
			}
			log.Printf("Client %v operation (doOps - %s) error: %s after %v microseconds", clientNum, opType, err.Error(), respTime)
			atomic.AddUint32(&e.stats.errors, 1)
			atomic.AddUint32(&e.intervalStats.errors, 1)
		} else {
			if resp.Put() != nil {
				putStatsMutex.Lock()
				e.stats.keysPut = e.stats.keysPut + 1
				e.intervalStats.keysPut = e.intervalStats.keysPut + 1
				e.stats.bytesPut = e.stats.bytesPut + int64(binary.Size(op.KeyBytes())) + int64(binary.Size(op.ValueBytes()))
				e.intervalStats.bytesPut = e.intervalStats.bytesPut + int64(binary.Size(op.KeyBytes)) + int64(binary.Size(op.ValueBytes()))
				e.stats.keysPutTotTime = e.stats.keysPutTotTime + respTime
				e.intervalStats.keysPutTotTime = e.intervalStats.keysPutTotTime + respTime
				if e.stats.keysPutMinTime == 0 || respTime < e.stats.keysPutMinTime {
					e.stats.keysPutMinTime = respTime
				}
				if e.intervalStats.keysPutMinTime == 0 || respTime < e.intervalStats.keysPutMinTime {
					e.intervalStats.keysPutMinTime = respTime
				}
				if e.stats.keysPutMaxTime == 0 || respTime > e.stats.keysPutMaxTime {
					e.stats.keysPutMaxTime = respTime
				}
				if e.intervalStats.keysPutMaxTime == 0 || respTime > e.intervalStats.keysPutMaxTime {
					e.intervalStats.keysPutMaxTime = respTime
				}
				putStatsMutex.Unlock()
				if verbose {
					log.Printf("Put completed for client %v after %v microseconds", clientNum, respTime)
				}
			} else if resp.Del() != nil {
				delStatsMutex.Lock()
				e.stats.clientDels = e.stats.clientDels + 1
				e.intervalStats.clientDels = e.intervalStats.clientDels + 1
				e.stats.keysDel = e.stats.keysDel + uint32(resp.Del().Deleted)
				e.intervalStats.keysDel = e.intervalStats.keysDel + uint32(resp.Del().Deleted)
				e.stats.keysDelTotTime = e.stats.keysDelTotTime + respTime
				e.intervalStats.keysDelTotTime = e.intervalStats.keysDelTotTime + respTime
				if e.stats.keysDelMinTime == 0 || respTime < e.stats.keysDelMinTime {
					e.stats.keysDelMinTime = respTime
				}
				if e.intervalStats.keysDelMinTime == 0 || respTime < e.intervalStats.keysDelMinTime {
					e.intervalStats.keysDelMinTime = respTime
				}
				if e.stats.keysDelMaxTime == 0 || respTime > e.stats.keysDelMaxTime {
					e.stats.keysDelMaxTime = respTime
				}
				if e.intervalStats.keysDelMaxTime == 0 || respTime > e.intervalStats.keysDelMaxTime {
					e.intervalStats.keysDelMaxTime = respTime
				}
				delStatsMutex.Unlock()
				if verbose {
					log.Printf("Delete completed for client %v after %v microseconds, %v keys were deleted", clientNum, respTime, uint32(resp.Del().Deleted))
				}
			} else if resp.Get() != nil {
				var bytesGet int64
				if fullGetRead {
					for _, kv := range resp.Get().Kvs {
						value := kv.Value
						key := kv.Key
						bytesGet = bytesGet + int64(len(value)) + int64(len(key))
						if verbose {
							fmt.Printf("Got (total len=%d), Key(len=%d): %s , Value(len=%d): %s \n", len(key)+len(value), len(key), key, len(value), value)
						}
					}
					// Want to include the time to read values in the resp time
					respTime = time.Since(st).Nanoseconds() / 1000
				}
				getStatsMutex.Lock()
				if prefixGets {
					e.stats.bytesPrefixGet = e.stats.bytesPrefixGet + bytesGet
					e.intervalStats.bytesPrefixGet = e.intervalStats.bytesPrefixGet + bytesGet
					e.stats.clientPrefixGets = e.stats.clientPrefixGets + 1
					e.intervalStats.clientPrefixGets = e.intervalStats.clientPrefixGets + 1
					e.stats.keysPrefixGet = e.stats.keysPrefixGet + uint32(resp.Get().Count)
					e.intervalStats.keysPrefixGet = e.intervalStats.keysPrefixGet + uint32(resp.Get().Count)
					e.stats.keysPrefixGetTotTime = e.stats.keysPrefixGetTotTime + respTime
					e.intervalStats.keysPrefixGetTotTime = e.intervalStats.keysPrefixGetTotTime + respTime
					if e.stats.keysPrefixGetMinTime == 0 || respTime < e.stats.keysPrefixGetMinTime {
						e.stats.keysPrefixGetMinTime = respTime
					}
					if e.intervalStats.keysPrefixGetMinTime == 0 || respTime < e.intervalStats.keysPrefixGetMinTime {
						e.intervalStats.keysPrefixGetMinTime = respTime
					}
					if e.stats.keysPrefixGetMaxTime == 0 || respTime > e.stats.keysPrefixGetMaxTime {
						e.stats.keysPrefixGetMaxTime = respTime
					}
					if e.intervalStats.keysPrefixGetMaxTime == 0 || respTime > e.intervalStats.keysPrefixGetMaxTime {
						e.intervalStats.keysPrefixGetMaxTime = respTime
					}
				} else {
					e.stats.bytesGet = e.stats.bytesGet + bytesGet
					e.intervalStats.bytesGet = e.intervalStats.bytesGet + bytesGet
					e.stats.clientGets = e.stats.clientGets + 1
					e.intervalStats.clientGets = e.intervalStats.clientGets + 1
					e.stats.keysGet = e.stats.keysGet + uint32(resp.Get().Count)
					e.intervalStats.keysGet = e.intervalStats.keysGet + uint32(resp.Get().Count)
					e.stats.keysGetTotTime = e.stats.keysGetTotTime + respTime
					e.intervalStats.keysGetTotTime = e.intervalStats.keysGetTotTime + respTime
					if e.stats.keysGetMinTime == 0 || respTime < e.stats.keysGetMinTime {
						e.stats.keysGetMinTime = respTime
					}
					if e.intervalStats.keysGetMinTime == 0 || respTime < e.intervalStats.keysGetMinTime {
						e.intervalStats.keysGetMinTime = respTime
					}
					if e.stats.keysGetMaxTime == 0 || respTime > e.stats.keysGetMaxTime {
						e.stats.keysGetMaxTime = respTime
					}
					if e.intervalStats.keysGetMaxTime == 0 || respTime > e.intervalStats.keysGetMaxTime {
						e.intervalStats.keysGetMaxTime = respTime
					}
				}
				getStatsMutex.Unlock()
				if verbose {
					log.Printf("Get completed for client %v after %v microseconds, %v keys were retrieved", clientNum, respTime, resp.Get().Count)
				}
			}
		}
	}
}
