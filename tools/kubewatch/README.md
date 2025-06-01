# Tools for debugging problems with cruiser creation

## Using Kubewatch to understand cruiser creation

[Kubewatch](https://github.ibm.com/nrockwell/kubewatch/blob/master/cmd/performance.go) is a Kubernetes watcher that currently publishes notification to available collaboration hubs/notification channels. This fork of [original kubewatch](https://github.com/bitnami-labs/kubewatch) has a performance channel that dumps basic information on kube events to a text file.  

Ex:
```
2020-07-10T15:57:29 master-bs3sojg20rlmlikosgn0/cluster-policy-controller-6dcc458785-cgvq8 pod updated Pending [{PodScheduled False 0/135 nodes are available: 16 node(s) had taints that the pod didn't tolerate, 35 Insufficient pods, 84 node(s) didn't match pod affinity/anti-affinity, 84 node(s) didn't satisfy existing pods anti-affinity rules.}]
2020-07-10T15:57:29 master-bs3dcn120roucc8tg2eg/cluster-policy-controller-65f495746c-cmtn7 pod updated Pending [{PodScheduled False 0/135 nodes are available: 16 node(s) had taints that the pod didn't tolerate, 35 Insufficient pods, 84 node(s) didn't match pod affinity/anti-affinity, 84 node(s) didn't satisfy existing pods anti-affinity rules.}]
2020-07-10T15:57:29 master-bs3mhu720t2atfdt1en0/openshift-apiserver-85bbd69db9-swj8v pod updated Pending [{PodScheduled False 0/135 nodes are available: 16 node(s) had taints that the pod didn't tolerate, 35 Insufficient pods, 84 node(s) didn't match pod affinity/anti-affinity, 84 node(s) didn't satisfy existing pods anti-affinity rules.}]
2020-07-10T15:57:30 master-bs48qve20n5m4rk8j7ng/manifests-bootstrapper pod updated Running False 10.209.135.72
2020-07-10T15:57:30 master-bs48rvj20knl5b48j7o0/openshift-apiserver-88978cff6-fkkjb pod updated Running False 10.74.176.241
```

Scripts take this output and convert it into a timeline useful for getting an overview of, for example, cruiser creation from a kubernetes perspective. This is particularly useful because with ROKS cluster because all operations are within a single namespace, which acts as a filter on events.

### Configuration
Kubewatch reads `~/.kubewatch.yaml` to determine  what vents are watched and where the events are sent.  

The configuration below enables the performance channel and specifies the resources that will be watched. Note that the "filter" feature of the performance configuration hasn't been implemented. 
```
handler:
  .....
  performance:
    flag: always
    filter: ""
resource:
  deployment: true
  replicationcontroller: false
  replicaset: true
  daemonset: false
  services: true
  pod: true
  job: false
  persistentvolume: false
  namespace: true
  secret: false
  configmap: true
  ingress: false
  endpoints: true
namespace: ""
```

### Watch kube events
Kubewatch should be run on a performance client. In the past stage-dal09-perf5-client-02 was used to watch carrier500.
```
export KUBECONFIG=/performance/config/carrier500_stage/admin-kubeconfig
cd /performance/stats/tugboat500
cp .kubewatch.yaml ~
nohup ./kubewatch &
```

### Run the following command from /performance/stats/tugboat500 to see timeline of cluster creation
```
Usage: createTimeline.sh <cluster id> [<name of log file to search. Defaults to nohup.out>]

Outputs:
<create time>,<running time for pod, creation time for other types>,<name>:<pod|namespace|service|deployment>
```

Ex:

```
$ ./createTimeline.sh bpj882i202c6hchmkqa0
2020-03-09 18:01:36,2020-03-09 18:01:36,master-bpj882i202c6hchmkqa0:namespace
2020-03-09 18:01:42,2020-03-09 18:01:42,kube-apiserver:service
2020-03-09 18:01:45,2020-03-09 18:01:45,openshift-apiserver:service
2020-03-09 18:03:52,2020-03-09 18:03:52,oauth-openshift:service
2020-03-09 18:04:09,2020-03-09 18:04:18,etcd-operator:deployment
2020-03-09 18:04:10,2020-03-09 18:04:10,etcd-restore-operator:service
2020-03-09 18:04:14,2020-03-09 18:04:17,etcd-operator-7df6b99b44-lcjxg:pod
2020-03-09 18:04:14,2020-03-09 18:04:18,etcd-operator-7df6b99b44-c9zf6:pod
2020-03-09 18:04:14,2020-03-09 18:04:18,etcd-operator-7df6b99b44-jn4tw:pod
2020-03-09 18:04:17,2020-03-09 18:04:17,etcd-bpj882i202c6hchmkqa0-client:service
2020-03-09 18:04:17,2020-03-09 18:04:17,etcd-bpj882i202c6hchmkqa0:service
2020-03-09 18:04:17,2020-03-09 18:04:24,etcd-bpj882i202c6hchmkqa0-5jjv48jnlh:pod
2020-03-09 18:04:31,2020-03-09 18:04:37,ca-operator:deployment
2020-03-09 18:04:32,2020-03-09 18:04:37,ca-operator-7cd4bc7785-gm7vf:pod
2020-03-09 18:04:32,2020-03-09 18:04:38,cluster-policy-controller-6f54445869-sfzjx:pod
2020-03-09 18:04:32,2020-03-09 18:04:38,cluster-version-operator-675b78f75d-zbfz9:pod
2020-03-09 18:04:32,2020-03-09 18:04:39,cluster-version-operator:deployment
2020-03-09 18:04:32,2020-03-09 18:04:44,cluster-policy-controller:deployment
2020-03-09 18:04:32,2020-03-09 18:04:46,kube-controller-manager:deployment
2020-03-09 18:04:32,2020-03-09 18:06:33,kube-apiserver:deployment
2020-03-09 18:04:33,2020-03-09 18:04:39,kube-apiserver-6c65d955d9-7zhmb:pod
2020-03-09 18:04:33,2020-03-09 18:04:42,kube-scheduler:deployment
2020-03-09 18:04:33,2020-03-09 18:04:55,oauth-openshift:deployment
2020-03-09 18:04:34,2020-03-09 18:04:39,cluster-policy-controller-6f54445869-tctpt:pod
2020-03-09 18:04:34,2020-03-09 18:04:39,kube-apiserver-6c65d955d9-6nw9d:pod
2020-03-09 18:04:34,2020-03-09 18:04:39,kube-apiserver-6c65d955d9-cc78v:pod
2020-03-09 18:04:34,2020-03-09 18:04:39,kube-controller-manager-59565bbcfb-pcsff:pod
2020-03-09 18:04:34,2020-03-09 18:04:41,kube-scheduler-6bfdb8845d-w25cn:pod
2020-03-09 18:04:34,2020-03-09 18:04:43,cluster-policy-controller-6f54445869-zxjvq:pod
2020-03-09 18:04:34,2020-03-09 18:04:47,openshift-apiserver:deployment
2020-03-09 18:04:35,2020-03-09 18:04:39,kube-scheduler-6bfdb8845d-rjz4s:pod
2020-03-09 18:04:35,2020-03-09 18:04:40,kube-controller-manager-59565bbcfb-7w4g7:pod
2020-03-09 18:04:35,2020-03-09 18:04:40,kube-scheduler-6bfdb8845d-znqvc:pod
2020-03-09 18:04:35,2020-03-09 18:04:42,openshift-apiserver-7c87fccddf-crwp2:pod
2020-03-09 18:04:35,2020-03-09 18:04:44,openshift-apiserver-7c87fccddf-r944l:pod
2020-03-09 18:04:35,2020-03-09 18:04:45,openshift-controller-manager:deployment
2020-03-09 18:04:35,2020-03-09 18:04:46,kube-controller-manager-59565bbcfb-2hws9:pod
2020-03-09 18:04:35,2020-03-09 18:04:47,openshift-apiserver-7c87fccddf-7cnm8:pod
2020-03-09 18:04:35,2020-03-09 18:04:55,oauth-openshift-59d789d794-8xjmw:pod
2020-03-09 18:04:35,2020-03-09 18:04:55,oauth-openshift-59d789d794-9qtg6:pod
2020-03-09 18:04:35,2020-03-09 18:04:55,oauth-openshift-59d789d794-qpn96:pod
2020-03-09 18:04:36,2020-03-09 18:04:42,openshift-controller-manager-578bb45668-98ph4:pod
2020-03-09 18:04:36,2020-03-09 18:04:44,openshift-controller-manager-578bb45668-j9569:pod
2020-03-09 18:04:36,2020-03-09 18:04:44,openshift-controller-manager-578bb45668-qs558:pod
2020-03-09 18:04:39,2020-03-09 18:05:13,etcd-bpj882i202c6hchmkqa0-jgrlbbjmrh:pod
2020-03-09 18:04:45,2020-03-09 18:04:50,manifests-bootstrapper:pod
2020-03-09 18:05:31,2020-03-09 18:06:04,etcd-bpj882i202c6hchmkqa0-wr4d6vwssw:pod
2020-03-09 18:07:43,2020-03-09 18:07:49,openvpn-operator:deployment
2020-03-09 18:07:45,2020-03-09 18:07:49,openvpn-operator-6769b995b8-hnw7h:pod
2020-03-09 18:07:47,2020-03-09 18:07:49,cloud-controller-manager-98f5955dd-2tnxt:pod
2020-03-09 18:07:47,2020-03-09 18:07:50,cloud-controller-manager-98f5955dd-bxlnh:pod
2020-03-09 18:07:47,2020-03-09 18:07:50,cloud-controller-manager-98f5955dd-kbrpn:pod
2020-03-09 18:07:47,2020-03-09 18:07:51,cloud-controller-manager:deployment
2020-03-09 18:07:51,2020-03-09 18:07:51,openvpn-operator-metrics:service
2020-03-09 18:08:10,2020-03-09 18:08:10,openvpnserver-managed-ovpn:service
2020-03-09 18:10:19,2020-03-09 18:10:24,cluster-health:deployment
2020-03-09 18:10:20,2020-03-09 18:10:24,cluster-health-7547989577-svjgs:pod
2020-03-09 18:15:49,2020-03-09 18:15:52,openvpnserver-managed-ovpn-5d67db8f98-xxvgm:pod
2020-03-09 18:15:49,2020-03-09 18:15:52,openvpnserver-managed-ovpn:deployment
2020-03-09 18:16:02,2020-03-09 18:16:05,kubx-etcd-backup-bpj882i202c6hchmkqa0-update-xzg29:pod
2020-03-09 18:16:40,2020-03-09 18:17:24,kube-apiserver-7c76d68c7c-59x5t:pod
2020-03-09 18:16:40,2020-03-09 18:18:27,kube-apiserver-7c76d68c7c-xnrff:pod
2020-03-09 18:16:40,2020-03-09 18:19:39,kube-apiserver-7c76d68c7c-gjswb:pod
2020-03-09 18:16:56,2020-03-09 18:16:59,cluster-updater-bpj882i202c6hchmkqa0-7f8bc97d4b-5fhxf:pod
2020-03-09 18:16:56,2020-03-09 18:16:59,cluster-updater-bpj882i202c6hchmkqa0:deployment
2020-03-09 18:17:22,2020-03-09 18:17:25,cluster-updater-bpj882i202c6hchmkqa0-6476d85c7b-xnvn9:pod
```

## Other scripts

### Watch for long running kubx-deployer pods
Collects data on long running kubx-deployer pods because the data is often gone before the issue is noticed.
```
cd /performance/stats/tugboat500/kubx-deployer
nohup ../watchForLongKubxDeploys.sh &
```

### Save etcd pods logs for all clusters. Waits for namespace to be at least 10 minutes old.
Useful to catch data from when the occasional  etcd cluster isn't constructed correctly.
```
cd /performance/stats/tugboat500/etcd
nohup ../watchEtcdOperatorLogs.sh &
```

### Scan ouptut of kubewatch to find how many apiservers have been created for each cruiser
```
./findClusterKubeApiCounts.sh
```

