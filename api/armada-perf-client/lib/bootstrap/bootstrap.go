/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	config "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/config"
	util "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/util"
	etcdV3Client "go.etcd.io/etcd/client/v3"
)

type worker struct {
	workerID  string
	workerKey string
}

// MockBootstrap provides a dummy armada bootstrap service.
type MockBootstrap struct {
	sync.RWMutex
	clientV3    *etcdV3Client.Client
	conf        *config.Config
	workers     map[string][]worker
	workerCount int
	verbose     bool
}

const bootstrapping string = "bootstrapping"

// InitMockBootstrap returns a MockDeploy instance.
func InitMockBootstrap(workerCount int, conf *config.Config, verbose bool) *MockBootstrap {
	etcdClientV3 := config.InitEtcdV3Client(conf.Etcd)

	workers := make(map[string][]worker)

	return &MockBootstrap{clientV3: etcdClientV3, conf: conf, workers: workers, workerCount: workerCount, verbose: verbose}
}

func (mockBootstrap *MockBootstrap) bootstrapWorker(clusterID string, workerIndex int, bootstrapped chan int) {
	const timeFormat = "2006-01-02T15:04:05-0700" // See https://github.ibm.com/alchemy-containers/armada-bootstrap/blob/master/lib/etcd.go

	mockBootstrap.RLock()
	var bootstrapStartDate string
	var bootstrapEndDate string
	var workerBootstrapped string

	bootstrapStartDate = strings.Join([]string{mockBootstrap.workers[clusterID][workerIndex].workerKey, "bootstrap_start_date", clusterID, mockBootstrap.workers[clusterID][workerIndex].workerID}, "/")
	bootstrapEndDate = strings.Join([]string{mockBootstrap.workers[clusterID][workerIndex].workerKey, "bootstrap_end_date", clusterID, mockBootstrap.workers[clusterID][workerIndex].workerID}, "/")
	workerBootstrapped = strings.Join([]string{mockBootstrap.workers[clusterID][workerIndex].workerKey, "bootstrap_status", clusterID, mockBootstrap.workers[clusterID][workerIndex].workerID}, "/")

	// Wait for indication that we're bootstrapping
	rch := mockBootstrap.clientV3.Watch(context.Background(), strings.Join([]string{mockBootstrap.workers[clusterID][workerIndex].workerKey, "state", clusterID, mockBootstrap.workers[clusterID][workerIndex].workerID}, "/"))
	mockBootstrap.RUnlock()

BootstrappingLoop:
	//		for wresp := range rch {
	for {
		timeout := make(chan bool, 1)
		go func() {
			time.Sleep(2 * time.Second)
			timeout <- true
		}()
		select {
		case <-timeout:
			// Just drop through and do the check
		case wresp, ok := <-rch:
			if !ok || wresp.Err() != nil {
				mockBootstrap.RLock()
				// Error occurred
				if strings.Contains(wresp.Err().Error(), "The event in requested index is outdated") {
					// Index from when watcher was created is too old - rereate the watcher
					if mockBootstrap.verbose {
						fmt.Printf("%s : Hit outdated index for %s , recreated watcher \n", time.Now().Format(time.StampMilli), mockBootstrap.workers[clusterID][workerIndex].workerID)
					}
					rch = mockBootstrap.clientV3.Watch(context.Background(), strings.Join([]string{mockBootstrap.workers[clusterID][workerIndex].workerKey, "state", clusterID, mockBootstrap.workers[clusterID][workerIndex].workerID}, "/"))
					mockBootstrap.RUnlock()
					continue
				}
				//Unexpected error - we might aswell try again, but log the error
				fmt.Printf("%s : Unexpected Watcher error for %s , the error was %s \n", time.Now().Format(time.StampMilli), mockBootstrap.workers[clusterID][workerIndex].workerID, wresp.Err().Error())
				mockBootstrap.RUnlock()
			}
		}
		mockBootstrap.RLock()
		state, err := util.V3GetWithRetry(mockBootstrap.clientV3, *mockBootstrap.conf, strings.Join([]string{mockBootstrap.workers[clusterID][workerIndex].workerKey, "state", clusterID, mockBootstrap.workers[clusterID][workerIndex].workerID}, "/"))
		if err != nil || state.Count < 1 {
			if mockBootstrap.verbose {
				fmt.Printf("%s : Watcher timed out or error for %s and state does not currently exist \n", time.Now().Format(time.StampMilli), mockBootstrap.workers[clusterID][workerIndex].workerID)
			}
			mockBootstrap.RUnlock()
			continue
		}

		if mockBootstrap.verbose {
			// Log the worker state change
			fmt.Printf("%s : %s %s\n", time.Now().Format(time.StampMilli), string(state.Kvs[0].Value), mockBootstrap.workers[clusterID][workerIndex].workerID)
		}
		mockBootstrap.RUnlock()

		if string(state.Kvs[0].Value) == bootstrapping {
			break BootstrappingLoop
		}

	}

	if _, err := util.V3PutWithRetry(mockBootstrap.clientV3, *mockBootstrap.conf, bootstrapStartDate, time.Now().Format(timeFormat)); err != nil {
		panic(err)
	}

	// Add in a delay if required
	delay := mockBootstrap.conf.Bootstrap.BootstrapWorkerDelay.Duration
	if delay > 0 {
		time.Sleep(delay)
	}

	if _, err := util.V3PutWithRetry(mockBootstrap.clientV3, *mockBootstrap.conf, workerBootstrapped, "COMPLETE"); err != nil {
		panic(err)
	}

	if _, err := util.V3PutWithRetry(mockBootstrap.clientV3, *mockBootstrap.conf, bootstrapEndDate, time.Now().Format(timeFormat)); err != nil {
		panic(err)
	}

	bootstrapped <- workerIndex
}

// PerformBootstrap mimics the action of aramda bootstrap to indicate that a worker has been bootstrapped
func (mockBootstrap *MockBootstrap) PerformBootstrap(action config.ActionType, accountID string, clusterID string, freeAccount bool) {
	var workerPrefix string

	if freeAccount {
		workerPrefix = strings.Join([]string{mockBootstrap.conf.Location.Environment, mockBootstrap.conf.Location.Datacenter, "pa"}, "-")
	} else {
		workerPrefix = strings.Join([]string{mockBootstrap.conf.Location.Environment, mockBootstrap.conf.Location.Datacenter, "cr"}, "-")
	}

	var existingWorkers = 0
	if action == config.ActionAddClusterWorkers {
		// When adding worker(s) to an existing cruiser we are given a cruiser name rather than an ID :(
		// So, time to look up the id from etcd
		c, err := util.V3GetWithRetry(mockBootstrap.clientV3, *mockBootstrap.conf, "/"+strings.Join([]string{mockBootstrap.conf.Location.Region, "accounts", accountID, "clusters", "all"}, "/"))
		if err != nil {
			panic(err)
		}

		var dat map[string]interface{}
		if err = json.Unmarshal(c.Kvs[0].Value, &dat); err != nil {
			panic(err)
		}

		clusterID = dat[clusterID].(string)

		w, err := util.V3GetWithRetry(mockBootstrap.clientV3, *mockBootstrap.conf, "/"+strings.Join([]string{mockBootstrap.conf.Location.Region, "rec/desired/clusters", clusterID, "workers"}, "/"), etcdV3Client.WithPrefix())
		if err != nil {
			panic(err)
		}
		existingWorkers = len(w.Kvs)/6 - mockBootstrap.workerCount

	}

	// Channel to wait for all workers to complete bootstrapping
	bootstrapped := make(chan int, mockBootstrap.workerCount)

	for workerIndex := 0; workerIndex < mockBootstrap.workerCount; workerIndex++ {
		workerID := fmt.Sprintf("%s%s-w%d", workerPrefix, clusterID, workerIndex+existingWorkers+1)
		var workerKey string
		if mockBootstrap.conf.Etcd.EtcdVersion == 2 { // ETCD V2 API
			workerKey = "/" + strings.Join([]string{mockBootstrap.conf.Location.Region, "actual/clusters", clusterID, "workers", workerID}, "/")
		} else {
			// For V3 the structure is different so can't construct whole key here
			workerKey = "/" + strings.Join([]string{mockBootstrap.conf.Location.Region, "actual/clusters/workers"}, "/")
		}

		tmpWorker := worker{workerID: workerID, workerKey: workerKey}
		mockBootstrap.Lock()
		mockBootstrap.workers[clusterID] = append(mockBootstrap.workers[clusterID], tmpWorker)
		mockBootstrap.Unlock()

		go mockBootstrap.bootstrapWorker(clusterID, workerIndex, bootstrapped)
	}

	for workerIndex := 0; workerIndex < mockBootstrap.workerCount; workerIndex++ {
		workerNum := <-bootstrapped
		if mockBootstrap.verbose {
			mockBootstrap.RLock()
			fmt.Printf("%s : %s bootstrapping completed.\n", time.Now().Format(time.StampMilli), mockBootstrap.workers[clusterID][workerNum].workerID)
			mockBootstrap.RUnlock()
		}
	}

	close(bootstrapped)
}
