/*******************************************************************************
 * IBM Confidential
 * OCO Source Materials
 * IBM Cloud Kubernetes Service, 5737-D43
 * (C) Copyright IBM Corp. 2017, 2020 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package cmd

//import "fmt"

// NewArmadaEngine crease the pattern engine for driving load
func NewArmadaEngine(mods map[string]int) *PatternEngine {
	for id, val := range mods {
		t := armadaPathRules[id]
		t.cnt = val
		armadaPathRules[id] = t
		//fmt.Printf("armadaPathRules[%s]=%v\n",id,armadaPathRules[id])
	}
	engine := NewPatternEngine(armadaPathRules, armadaPatterns)
	return engine
}

var armadaPathRules = map[string]patternRule{
	"actual_desired": {nodeType: TypePatterns, patterns: []string{"actual", "desired"}, max: 3, cnt: 2},
	"clusterid":      {nodeType: TypePattern, pattern: "[0-9]{2}[a-f0-9]{37}", max: 0, cnt: 1},
	"masterid":       {nodeType: TypePattern, pattern: "[a-f0-9]{37}", max: 0, cnt: 1},
	"region": {nodeType: TypePatterns,
		patterns: []string{"us-south", "us-east", "eu-frank", "au-south"}, max: 4, cnt: 4},
	"state": {nodeType: TypePatterns,
		patterns: []string{"unprovisioned", "provisioning", "provision_failed", "provisioned", "deploying", "deploy_failed", "deployed", "deleting", "delete_failed", "deletefailed", "deleted"}, max: 11, cnt: 11},
	"workerid": {nodeType: TypePattern, pattern: "dev-mex01-[a-f0-9]{37}", max: 0, cnt: 1},
	"ip":       {nodeType: TypePatterns, patterns: []string{getIP()}, max: 1, cnt: 1}, // Allows for uniqe keys between pods
}

var armadaPatterns = []string{
	///us-south/actual/clusters/cr77a18e0926b64a488385cb98dd6837b0/workers/dev-mex01-cr77a18e0926b64a488385cb98dd6837b0-w1/state
	// TODO update size based on actual data
	"/:region/:actual_desired/clusters/:clusterid/name;[a-z0-9]{20}",       // new for 2017-02-06
	"/:region/:actual_desired/clusters/:clusterid/datacenter;[a-z0-9]{12}", // new for 2017-02-06
	"/:region/actual/clusters/:clusterid/ssh_key_public;[a-z0-9]{1000}",
	"/:region/actual/clusters/:clusterid/ssh_key_private;[a-z0-9]{725}",
	"/:region/actual/clusters/:clusterid/ssh_key_id;[a-z0-9]{20}",
	"/:region/:actual_desired/clusters/:clusterid/refresh_token;[a-z0-9]{50}",        // new for 2017-02-06
	"/:region/:actual_desired/clusters/:clusterid/api_user;[a-z0-9]{20}",             // new for 2017-02-06
	"/:region/:actual_desired/clusters/:clusterid/api_key;[a-z0-9]{50}",              // new for 2017-02-06
	"/:region/:actual_desired/clusters/:clusterid/registry_docker_cfg;[a-z0-9]{100}", // new for 2017-02-06
	"/:region/actual/clusters/:clusterid/ca_cert;[a-z0-9]{734}",
	"/:region/actual/clusters/:clusterid/admin_cert;[a-z0-9]{50}",
	"/:region/actual/clusters/:clusterid/admin_key;[a-z0-9]{20}",
	"/:region/actual/clusters/:clusterid/server_url;[a-z0-9]{30,100}",
	// TODO allow :state to be used for value
	"/:region/actual/clusters/:clusterid/state;(provisioning|provisioned|deploying|deployed|deleting|deletefailed|deleted)",
	"/:region/:actual_desired/clusters/:clusterid/workers/:workerid/state;(provisioning|provisioned|deploying|deployed|deleting|deletefailed|deleted)",
	"/:region/:actual_desired/clusters/:clusterid/workers/:workerid/priviate_vlan;[0-9]{10}",      // new for 2017-02-06
	"/:region/:actual_desired/clusters/:clusterid/workers/:workerid/public_vlan;[0-9]{10}",        // new for 2017-02-06
	"/:region/:actual_desired/clusters/:clusterid/workers/:workerid/unused_private_ips;[0-9]{10}", // new for 2017-02-06
	"/:region/:actual_desired/clusters/:clusterid/workers/:workerid/unused_public_ips;[0-9]{10}",  // new for 2017-02-06
	"/:region/:actual_desired/clusters/:clusterid/workers/:workerid/used_private_ips;[0-9]{10}",   // new for 2017-02-06
	"/:region/:actual_desired/clusters/:clusterid/workers/:workerid/unused_public_ips;[0-9]{10}",  // new for 2017-02-06
	"/:region/:actual_desired/clusters/:clusterid/workers/:workerid/machine_size;[0-9]{10}",
	"/:region/:actual_desired/clusters/:clusterid/workers/:workerid/datacenter;[a-z]{10}",
	"/:region/:actual_desired/clusters/:clusterid/workers/:workerid/machine_id;[0-9]{10}",
	"/:region/:actual_desired/clusters/:clusterid/workers/:workerid/public_ip;[0-9]{3}.[0-9]{3}.[0-9]{3}.[0-9]{3}",
	"/:region/:actual_desired/clusters/:clusterid/workers/:workerid/private_ip;[0-9]{3}.[0-9]{3}.[0-9]{3}.[0-9]{3}",
	"/:region/:actual_desired/clusters/:clusterid/workers/:workerid/machine_status;[a-z]{8}",
	"/:region/:actual_desired/clusters/:clusterid/workers/:workerid/os_status;[a-z]{8}",
	"/:region/:actual_desired/clusters/:clusterid/masters/:masterid/state;(provisioning|provisioned|deploying|deployed|deleting|deletefailed|deleted)",
}
