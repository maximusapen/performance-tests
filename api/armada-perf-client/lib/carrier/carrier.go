/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package carrier

import (
	"context"
	"strings"

	config "github.ibm.com/alchemy-containers/armada-performance/api/armada-perf-client/lib/config"
)

// InitCarrier sets up carrier etcd information
func InitCarrier(machineType string, conf *config.Config) {
	datacenterPrefix := "/" + strings.Join([]string{conf.Location.Region, "datacenters", conf.Location.Datacenter}, "/")
	machineTypeKeyPrefix := strings.Join([]string{datacenterPrefix, "machine_types", machineType}, "/")

	etcdClientV3 := config.InitEtcdV3Client(conf.Etcd)

	// Populate datacenters and machine sizes
	defer etcdClientV3.Close()

	ctx, cancel := context.WithTimeout(context.Background(), conf.Etcd.EtcdTimeout.Duration)
	if _, err := etcdClientV3.Put(ctx, strings.Join([]string{datacenterPrefix, "datacenter_id"}, "/"), "1234567"); err != nil {
		panic(err)
	}

	if _, err := etcdClientV3.Put(ctx, strings.Join([]string{machineTypeKeyPrefix, "name"}, "/"), machineType); err != nil {
		panic(err)
	}
	if _, err := etcdClientV3.Put(ctx, strings.Join([]string{machineTypeKeyPrefix, "server_type"}, "/"), "virtual"); err != nil {
		panic(err)
	}
	if _, err := etcdClientV3.Put(ctx, strings.Join([]string{machineTypeKeyPrefix, "max_memory"}, "/"), "64GB"); err != nil {
		panic(err)
	}
	if _, err := etcdClientV3.Put(ctx, strings.Join([]string{machineTypeKeyPrefix, "num_cores"}, "/"), "16"); err != nil {
		panic(err)
	}
	if _, err := etcdClientV3.Put(ctx, strings.Join([]string{machineTypeKeyPrefix, "network_speed"}, "/"), "1000Mbps"); err != nil {
		panic(err)
	}
	if _, err := etcdClientV3.Put(ctx, strings.Join([]string{machineTypeKeyPrefix, "operating_system"}, "/"), "UBUNTU_16_64"); err != nil {
		panic(err)
	}
	if _, err := etcdClientV3.Put(ctx, strings.Join([]string{machineTypeKeyPrefix, "primary_storage"}, "/"), "100GB"); err != nil {
		panic(err)
	}
	if _, err := etcdClientV3.Put(ctx, strings.Join([]string{machineTypeKeyPrefix, "os_item_code"}, "/"), "UBUNTU_16_64"); err != nil {
		panic(err)
	}
	if _, err := etcdClientV3.Put(ctx, strings.Join([]string{machineTypeKeyPrefix, "package_id"}, "/"), "801"); err != nil {
		panic(err)
	}
	if _, err := etcdClientV3.Put(ctx, strings.Join([]string{machineTypeKeyPrefix, "private_plan_id"}, "/"), "100"); err != nil {
		panic(err)
	}
	if _, err := etcdClientV3.Put(ctx, strings.Join([]string{machineTypeKeyPrefix, "public_plan_id"}, "/"), "101"); err != nil {
		panic(err)
	}
	cancel()
}
