/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package deploy

import (
	"fmt"
	"strings"
	"time"

	config "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/config"
	util "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/util"
	etcdV3Client "go.etcd.io/etcd/client/v3"
)

// MockDeploy provides a mock armada deploy service.
type MockDeploy struct {
	etcdClientV3 *etcdV3Client.Client
	conf         *config.Config
	verbose      bool
}

// InitMockDeploy returns a MockDeploy instance
func InitMockDeploy(conf *config.Config, verbose bool) *MockDeploy {
	etcdClientV3 := config.InitEtcdV3Client(conf.Etcd)
	return &MockDeploy{etcdClientV3: etcdClientV3, conf: conf, verbose: verbose}
}

// DeployCruiserMaster mimics the action of aramda deploy to indicate that the
// cruiser master is deployed and ready to accept workers.
func (mockDeploy *MockDeploy) DeployCruiserMaster(clusterID string) {
	if mockDeploy.verbose {
		fmt.Printf("%s : Cruiser Master %s deployment started.\n", time.Now().Format(time.StampMilli), clusterID)
	}
	var clusterKey string
	var artifactsKey string
	var etcdPemKey string
	var kubeletPemKey string
	var dockerRegistryKey string
	var masterDeployedKey string

	if mockDeploy.conf.Etcd.EtcdVersion == 2 {
		clusterKey = "/" + strings.Join([]string{mockDeploy.conf.Location.Region, "actual/clusters", clusterID}, "/")
		artifactsKey = strings.Join([]string{clusterKey, "artifacts"}, "/")
		etcdPemKey = strings.Join([]string{artifactsKey, "etcd-key.pem"}, "/")
		kubeletPemKey = strings.Join([]string{artifactsKey, "kubelet-key.pem"}, "/")
		dockerRegistryKey = strings.Join([]string{clusterKey, "registry_docker_cfg"}, "/")
		masterDeployedKey = strings.Join([]string{clusterKey, "ready_for_workers"}, "/")
	} else {
		clusterKey = "/" + strings.Join([]string{mockDeploy.conf.Location.Region, "actual/clusters"}, "/")
		artifactsKey = strings.Join([]string{clusterKey, "artifacts"}, "/")
		etcdPemKey = strings.Join([]string{artifactsKey, "etcd-key.pem", clusterID}, "/")
		kubeletPemKey = strings.Join([]string{artifactsKey, "kubelet-key.pem", clusterID}, "/")
		dockerRegistryKey = strings.Join([]string{clusterKey, "registry_docker_cfg", clusterID}, "/")
		masterDeployedKey = strings.Join([]string{clusterKey, "ready_for_workers", clusterID}, "/")
	}

	if _, err := util.V3PutWithRetry(mockDeploy.etcdClientV3, *mockDeploy.conf, etcdPemKey, "ToBeReplacedByMockEtcdPemKey"); err != nil {
		panic(err)
	}
	if _, err := util.V3PutWithRetry(mockDeploy.etcdClientV3, *mockDeploy.conf, kubeletPemKey, "ToBeReplacedByMockKubeletPemKey"); err != nil {
		panic(err)
	}
	if _, err := util.V3PutWithRetry(mockDeploy.etcdClientV3, *mockDeploy.conf, dockerRegistryKey, "docker_registry"); err != nil {
		panic(err)
	}
	// Add in a delay if required
	delay := mockDeploy.conf.Deploy.DeployMasterDelay.Duration
	if delay > 0 {
		time.Sleep(delay)
	}
	// Finally, indicate Cruiser Master deploy is complete
	if _, err := util.V3PutWithRetry(mockDeploy.etcdClientV3, *mockDeploy.conf, masterDeployedKey, "true"); err != nil {
		panic(err)
	}

	// Add in a delay if required
	if delay > 0 {
		time.Sleep(delay)
	}

	if mockDeploy.verbose {
		fmt.Printf("%s : Cruiser Master %s deployment completed.\n", time.Now().Format(time.StampMilli), clusterID)
	}
}
