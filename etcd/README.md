# etcd performance management tools

Etcd is a "distributed reliable key-value store for the most critical data of a distributed system" that is used as the database for Kubernetes and for Armada Microservices". For further details see [etcd](https://github.com/etcd-io/etcd).

## Layout of this repo

* etcd-backup: Primarily used for dumping the contents of etcd to a CSV file for examination
* etcd-benchmark: A copy of the community tool modified to allow specification of Armada Microservice like key/value pairs, as well as several additional tests.
* etcd-driver: A copy of a community tool to drive load against etcd, along with several helm charts including one to drive armada microservice like load. 
* etcd-operator: Scripts for creating an etcd cluster via [etcd-operator](https://github.com/coreos/etcd-operator). [Armada version of etcd-operator](https://github.ibm.com/alchemy-containers/etcd-operator)
* scripts: Scripts for creating etcd clusters, utilizing NFS for the database, via statefulsets. Not the best approach since etcd-operator was introduced. OLD
* test-harness: Scripts for setting up and managing the testing of an etcd cluster, optimally created via the armada etcd-operator cluster.
* etcd-rules-benchmark - Uses a benchmark provided by the armada-ballast team to measure the performance of the rules engine.

## Approach to testing etcd

The primary use for these tools is to test the armada microservices etcd database. The armada database exists in in a tugboat (ex. carrier501 for carrier5 and carrier500) and is managed by the etcd-operator that runs in that cluster. The etcd cluster runs on a series of dedicated nodes. Optimally, when testing etcd, the test cluster should be setup in the same tugboat with an existing armada etcd cluster. This ensures that the test setup is as close to possible to a production environment as possible. For best results additional nodes, dedicated to etcd, may need to be provisioned in order that each etcd pod runs alone on a node. This was done for example in carrier501. Assuming this approach is taken then take a look at test-harness for the setup and management of the cluster and either etcd-benchmark or etcd-driver for driving the tests.

Example steps:
1. `cd test-harness`
1. Edit etcd-perftest-config to reflect the configuration you want deployed. In this example the cluster was named 'etcd-5node'.
1. `nohup ./deploy-etcd.sh &`
1. ./get-etcd-endpoint-status.sh`  -> Check the status of the cluster
1. `cp etcd-perftest-config ../etcd-driver/imageDeploy`
1. `cd ../etcd-driver/imageDeploy`
1. `./deploy_etcd_driver.sh etcd-5node armada armada-etcd-simulator`
1. The default deploy can take 30-45 minutes to setup, so wait at least 2 hours before stopping the test
1. `./stop_test_collect_results.sh etcd-5node armada armada-etcd-simulator`
1. Process the results of the test run (see [etcd-driver/imageDeploy](./etcd-driver/imageDeploy/README.md))
1. `./delete_etcd_driver.sh etcd-5node armada armada-etcd-simulator`    -> The etcd-driver pods remain until deleted, though they won't generate load after test is stopped. The exception is if nodes are rebooted, in which case they will start generating load.

Alternative testing methodologies include:
* Creating a local etcd cluster on any host or set of hosts. Using dedicated hosts can be useful for removing the noise from tests compared to running against a cloud environment (i.e. noisy neighbor, etc.). (See scripts in [./etcd-benchmark](./etcd-benchmark))
* Create a etcd cluster with a set of pods (See scripts in [./scripts](.scripts)).

Results of previous tests can be found in [Box](https://ibm.ent.box.com/folder/19773796269)

# Nuggets 
* Running the test etcd cluster in the same environment as the armada etcd allows for comparison of the logs from both clusters to more clearly discern the root cause of an issue. For example, if the heavily loaded test cluster and the lightly used armada cluster show similar problems with in cluster connectivity, then most likely the problem has nothing to do with the test. In one case this showed that the network was struggling and impacting both clusters.
* There are several different service endpoints exposed to give access to etcd clusters when they are created via [./test-harness](./test-harness):
  * Cluster IP - Available to pods within the same namespaces (Create by etcd-operator)
  * Node Port - Available outside of kubernetes
  * Load Balancer - Available outside of kubernetes
  * Virutal IP - Used by armada microservices to access armada etcd. Also available to etcd-operator created clusters within the tugboat where armada etcd resides. The node port is specified.
  ```
  $ k -n armada get svc -o wide | grep 5nod
  etcd-5node                                       ClusterIP      None             <none>           2379/TCP,2380/TCP   5d22h   app=etcd,etcd_cluster=etcd-5node
  etcd-5node-client                                ClusterIP      172.19.42.234    <none>           2379/TCP            5d22h   app=etcd,etcd_cluster=etcd-5node
  etcd-5node-client-service-lb                     LoadBalancer   172.19.162.237   169.61.187.38    2379:31106/TCP      5d22h   etcd_cluster=etcd-5node
  etcd-5node-client-service-np                     NodePort       172.19.120.73    <none>           2379:30071/TCP      5d22h   etcd_cluster=etcd-5node
  ```
* Tugboat nodes are rebooted regularly. This causes the pods in both the etcd test cluster, as well as the load driver, to be moved to a different node. For etcd this just means the losss of the etcd logs. For etcd-driver the logs are lost, but more importantly the accumulated statistics on the load are lost. For the time being it is best to just restart the test. Long term it would help speed up testing if the etcd-driver statistics were stored outside of the pods (ex: the metrics server or a persistent volume).

