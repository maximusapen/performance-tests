/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2021 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package util

import (
	"context"
	"log"

	config "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/config"
	etcdV3Client "go.etcd.io/etcd/client/v3"
)

// V3PutWithRetry Utility method to retry an etcd Put using values in the perf.toml file
func V3PutWithRetry(cli *etcdV3Client.Client, conf config.Config, key string, value string) (resp *etcdV3Client.PutResponse, err error) {
	var savedErr error
	for i := 1; i <= conf.Etcd.EtcdRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), conf.Etcd.EtcdTimeout.Duration)
		resp, err := cli.Put(ctx, key, value)
		cancel()
		if err != nil {
			savedErr = err
			log.Printf("Etcd Put operation error for key %s : %s", key, err.Error())
			if (i + 1) < conf.Etcd.EtcdRetries {
				log.Printf("Attempt %v of %v ", i, conf.Etcd.EtcdRetries)
			}
		} else {
			return resp, nil
		}
	}
	log.Printf("Failed etcd operation after %v retries", conf.Etcd.EtcdRetries)
	return nil, savedErr
}

// V3GetWithRetry Utility method to retry an etcd Gut using values in the perf.toml file
func V3GetWithRetry(cli *etcdV3Client.Client, conf config.Config, key string, getOpts ...etcdV3Client.OpOption) (resp *etcdV3Client.GetResponse, err error) {
	var getErr error
	var getResp *etcdV3Client.GetResponse
	for i := 1; i <= conf.Etcd.EtcdRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), conf.Etcd.EtcdTimeout.Duration)

		getResp, getErr = cli.Get(ctx, key, getOpts...)

		cancel()
		if err != nil {
			log.Printf("Etcd Get operation error for key %s: %s", key, err.Error())
			if (i + 1) < conf.Etcd.EtcdRetries {
				log.Printf("Attempt %v of %v ", i, conf.Etcd.EtcdRetries)
			}
		} else {
			return getResp, nil
		}
	}
	log.Printf("Failed etcd operation after %v retries", conf.Etcd.EtcdRetries)
	return nil, getErr
}
