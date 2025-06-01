# Cruiser Metric Collection Program

This cruiser-collector utility can be used to gather CPU and Memory data from a Cluster, and optionally publish to the IBM Cloud Monitoring Serivce.

Data is collected via the Kubernetes Metrics Server and as such requires a cluster running Kubernetes 1.12 or above.

The utility is controlled by sending control commands to a network socket listening on a control port. Each "command" is a single byte as follows:

'0x1' START
* Start collecting metrics after any specified delay period  

'0x2' STOP
* Stop collecting metrics  

'0x3' TERMINATE
* End collection utility

#### Example
  * linux
    * `echo -en "\x1"  | nc localhost 20569`  
  * macos
    * `printf %b \\1 | nc localhost 20569`

## How to run

#### Parameters:
  * `-kubeconfig` : path to admin-kubeconfig file
  * `-delay` : time before data collection starts. Default is no delay.
  * `-interval` : time between the collection of each set of metrics. Default interval is 60s.
  * `-controlPort` : port on which to listen for control commands. Default port is 20569
  * `-filter` : regular expression to limit metrics published by name. Default is no filter.
  * `-level` : level at which metrics should be published. Default is to aggregate at pod level.
    * **node** - Node metrics only
    * **namespace** - Node and Pod metrics aggregated at the Namespace level
    * **pod** - Node and Pod metrics aggregated at the Pod level
    * **container** - Node and Pod metrics collected at the Container level
  * `-test`: name of test to be included in published metrics name. Default is not to include a test name.
  * `-verbose` : output *all* metrics received from the metrics server to `stdout`. Default is no output.
  * `-publish` Publish results to the IBM Cloud Monitoring Service. Default is **not** to publish.


### Example usage :
`cruiser-collector -kubeconfig="~/carrier4_stgiks/admin-kubeconfig" -delay=30s -interval=60s -level=namespace -publish`

  * After a 30 second delay, metrics will be collected and published to the IBM Cloud Monitoring Service every 30 seconds. Published metrics will include cpu and memory utilization of :  

    * each node in the cluster
    * all worker nodes in the cluster aggregated at the namespace level.

### Output

**Verbose**  

* **Node**  
2018-10-12T13:58:32Z
  * 10.142.173.193 - CPU: 78186331n, Mem: 2080444Ki
  * 10.142.173.190 - CPU: 71692939n, Mem: 1580700Ki
  * 10.142.173.192 - CPU: 116795967n, Mem: 2244164Ki  

* **Namespace**  
2018-10-12T13:59:05Z  

  * ibm-system: ibm-cloud-provider-ip-169-54-253-6-75b94c446c-8h4zf
	* ibm-cloud-provider-ip-169-54-253-6 - CPU: 608719n, Mem: 6528Ki
 
  * kube-system: tiller-deploy-5589b649cf-q8hb4
	* tiller - CPU: 606597n, Mem: 8944Ki

  * kube-system: calico-kube-controllers-5597449999-s8b4j
	* calico-kube-controllers - CPU: 820u, Mem: 18000Ki  

  * kube-system: calico-node-gjqdt
	* calico-node - CPU: 17609391n, Mem: 22948Ki
	* install-cni - CPU: 181218n, Mem: 48012Ki  

  * kube-system: ibm-file-plugin-644c448b55-2nmp4
	* ibm-file-plugin-container - CPU: 271063n, Mem: 9440Ki  

  * kube-system: vpn-7bbf5cd776-9b84r
	* vpn - CPU: 97064n, Mem: 2228Ki  

  * kube-system: ibm-keepalived-watcher-7wj4k
	* keepalived-watcher - CPU: 21355n, Mem: 7896Ki  

  * kube-system: calico-node-7ldtn
	* install-cni - CPU: 171063n, Mem: 49340Ki
	* calico-node - CPU: 15734835n, Mem: 23440Ki  

  * kube-system: ibm-master-proxy-static-10.142.173.193
	* ibm-master-proxy-static - CPU: 1341610n, Mem: 6164Ki  

  * kube-system: kube-dns-autoscaler-587cd5cd44-248jv
	* autoscaler - CPU: 296968n, Mem: 8888Ki  

  * kube-system: kubernetes-dashboard-78bf89c5d8-x8hqh
	* kubernetes-dashboard - CPU: 4226577n, Mem: 13920Ki  

  * kube-system: ibm-master-proxy-static-10.142.173.190
	* ibm-master-proxy-static - CPU: 309344n, Mem: 6312Ki  

  * kube-system: ibm-storage-watcher-669fc9b545-9tz5c
	* ibm-storage-watcher-container - CPU: 641736n, Mem: 8848Ki  

  * kube-system: ibm-keepalived-watcher-swjqq
	* keepalived-watcher - CPU: 11050n, Mem: 8000Ki  

  * kube-system: ibm-kube-fluentd-l2nfd
	* fluentd - CPU: 5983080n, Mem: 99024Ki  

  * kube-system: kube-dns-amd64-755d648656-7brs2
	* kubedns - CPU: 688260n, Mem: 8440Ki
	* dnsmasq - CPU: 113019n, Mem: 6840Ki
	* sidecar - CPU: 1143578n, Mem: 12852Ki  

  * kube-system: metrics-server-65f478cf95-glvsl
	* metrics-server - CPU: 974984n, Mem: 20308Ki
	* metrics-server-nanny - CPU: 377062n, Mem: 9536Ki  

  * kube-system: ibm-master-proxy-static-10.142.173.192
	  * ibm-master-proxy-static - CPU: 1756496n, Mem: 8480Ki  

  * kube-system: ibm-keepalived-watcher-kxzg9
	  * keepalived-watcher - CPU: 0, Mem: 7988Ki  

  * kube-system: ibm-kube-fluentd-9mkh6
	 * fluentd - CPU: 8308208n, Mem: 110912Ki  

  * kube-system: ibm-kube-fluentd-dczs9
	* fluentd - CPU: 4849585n, Mem: 97104Ki  

  * ibm-system: ibm-cloud-provider-ip-169-54-253-6-75b94c446c-fz8qb
	* ibm-cloud-provider-ip-169-54-253-6 - CPU: 691846n, Mem: 5464Ki  

  * kube-system: calico-node-sl2ks
	* install-cni - CPU: 269406n, Mem: 48596Ki  
	* calico-node - CPU: 17615156n, Mem: 22692Ki  

  * kube-system: kube-dns-amd64-755d648656-2czr5
	* dnsmasq - CPU: 138134n, Mem: 6644Ki
	* kubedns - CPU: 612284n, Mem: 9500Ki
	* sidecar - CPU: 844633n, Mem: 13612Ki  

**Published**   

{Name, Timestamp(Unix time), Value(cpu in millicores, memory in bytes)}
   * **Namespace**  
[{cruiser.namespace-metrics.ibm-system.cpu.sparse-avg 1539352745 1.300565} {cruiser.namespace-metrics.ibm-system.mem.sparse-avg 1539352745 1.2279808e+07} {cruiser.namespace-metrics.kube-system.cpu.sparse-avg 1539352745 86.013756} {cruiser.namespace-metrics.kube-system.mem.sparse-avg 1539352745 7.42305792e+08}]  

  * **Node**  
[{cruiser.node-metrics.10.142.173.193.cpu.sparse-avg 1539352712 78.186331} {cruiser.node-metrics.10.142.173.193.mem.sparse-avg 1539352712 2.130374656e+09} {cruiser.node-metrics.10_142_173_190.cpu.sparse-avg 1539352713 71.692939} {cruiser.node-metrics.10.142.173.190.mem.sparse-avg 1539352713 1.6186368e+09} {cruiser.node-metrics.10_142_173_192.cpu.sparse-avg 1539352715 116.795967} {cruiser.node-metrics.10.142.173.192.mem.sparse-avg 1539352715 2.298023936e+09}]
  

