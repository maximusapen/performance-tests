# Cloud Monitoring Tool
Cloud_mon was written to test basic armada capabilities while performing CTO requested failure testing. It runs in a continuous loop performing these tests:
1. Is armada API accessible?
2. Is armada API functional (create a cruiser)
3. Is cruiser accessible (access kubernetes API)
4. Is cruiser functional (create a client application)
5. Is client app accessible (hit client app with a simple request)

Cloud_mon utilizes 
1. A cluster, named clusterNamePrefix1, which is created when cloud_mon is started if it doesn't already exist.
2. 3 pod etcd cluster running on clusterNamePrefix1 which is accessed for test #5. The service for the app must be named etcdcm-public and must be create with ../config/etcdcm.yml.
3. A cluster, name clusterNamePrefix2, which is created and deleted for test #2.
4. An application definition (-activeApp) which by default is defined by [cloud-mon-app.yml](../config/cloud-mon-app.yml). It is used for test #4.

## Initial Setup
Cloud_mon relies on GOPATH and the configuration data in armada-performance/api/config. In addition there must be a toml file name perf.toml in api/config. The toml file isn't checked in because it includes authentication data.

1. Run cloud_mon once in order to force it to create clusterNamePrefix1. It may error our if the creation took too long in which case you need to check that the cluster is fully deployed before moving to the next step. Even if the cluster is created on time it will error out because the background etcd cluster hasn't been created.
2. Create the background etcd cluster
```
KUBECONFIG=clusterNamePrefix1/kube.yml kubectl create -f ../config/etcdcm.yml
```

## Running Tests
By default simply run cloud_mon:

```
./cloud_mon
```

Depending on what you are trying to accomplish it may be best to run multiple instances of cloud_mon. This would be the case is your testing might block one of the tests cases and thus prevent the others from running on a regular basis. For example if creation of a new cluster may block if carrier master etcd isn't accessible. In this case use the '-tests' parameter to define the tests to be run.

```
./cloud_mon -tests "y,y,n,n,n"
./cloud_mon -tests "n,n,y,y,y"
```
