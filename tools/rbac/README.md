# RBAC
Tools to test RBAC operator at scale. The RBAC operator exists in each IKS and ROKS cluster. The tests were run with IKS clusters. Scripts that operate on etcd clusters won't work with ROCKS clusters since they assume etcd runs in `kubx-etcd-##` namespace.

Issue for which scripts were developed: https://github.ibm.com/alchemy-containers/armada-performance/issues/1799

The scripts are used to load RBAC users into clusters, monitor etcd impact and determine overhead of authorization.

## Setup
Several of the scripts rely on the existence of a cluster in clusters.txt.

    ibmcloud ks clusters > clusters.txt

## Scripts
Add the specified number of RBAC users to the specified cluster. It sets KUBECONFIG before calling createRBACUserObjects.sh, and records the time to load the users.

    ./setupRBACCluster.sh <cluster name> <number of users>

This script is used to test the impact of loading RBAC users concurrently into a set of clusters. It monitors `kubectl get svc` times for each cluster before, during and after the test. It was a manual operation to parse get svc times from logs and create statistics for before, during and after the test run.

    ./createRBACUserObjectsConcurrent.sh <start cruiser postfix number> <end cruiser postfix number> <number of users>

These scripts tests the performance of RBAC authentication for admin and non-admin users. Note that scripts assume the clusters are named `fakecruiser-churn-<number>`. setupMultipleRBACClusters.sh needs to be updated with existing clusters prior to being run.

    ./setupMultipleRBACClusters.sh
    nohup ./getEndUserAuthorizationTimes.sh <cluster name>
    ./parseEndUserAuthorizationTimes.sh nohup.out

Determine how much etcd db grows due to loading RBAC users. A compress/defrag cycle is run at several points to insure results highlight real db size. Assumes KUBECONFIG is set to carrier kube.

    ./runRBACClusterSizeTest.sh <cluster name> <number of users>

This script runs runRBACClusterSizeTest.sh for multiple clusters with different number of users each. Needs updating with existing cluster names. Assumes carrier KUBECONFIG is set.

    ./getMultipleDBSize.sh

Then at a later point in time you can get the database sizes for those clusters. Needs updating with existing cluster names.

    ./getMultipleDBSize.sh


The same as `runRBACClusterSizeTest.sh` except that no compress/defrag after the users are loaded. Assumes KUBECONFIG is set to carrier kube.

    ./runRBAClusterSizeTestNoPost.sh <cluster name> <number of users>

Determine how much etcd db grows due to loading RBAC users, and record etcd db size every hour for 24 hours. The etcd db was seen growing significantly after the users were added. This was due to operations on the new users, and resulted in a lot of obsolete data that would disappear after compress/defrag. The script doesn't do a compress/defrag after the users are added. Assumes KUBECONFIG is set to carrier kube.

    ./runRBACHourlyClusterSizeTest.sh <cluster name> <number of users>

## Helper Scripts

Loads the specified number of RBAC users into the cluster specified by the current kubeconfig.

    ./createRBACUserObjects.sh [<number of users:defaults to 5000>]

Runs etcd compaction on each etcd server and then defrag on the etcd cluster. Assumes carrier KUBECONFIG is set.

    ./compressDefragClusterEtcd.sh <cluster name>

Returns the kubx-etcd-## namespace where the cluster is located

    ./findetcdnamespaces.sh <cluster id>

Gets the database size of each etcd server in the etcd clusters

    ./getClusterEtcdData.sh <cluster name (must be "fakecruiser-churn-*")|cluster id>

Preloads fakecruiser-churn-# kubeconfigs

    ./loadKubeConfigs.sh <start cluster number> <end cluster number>


