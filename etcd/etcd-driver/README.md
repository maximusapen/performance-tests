# Etcd-driver benchmark application

Etcd-driver has three extensions  above and beyond the community version:
- armada:      Generate key/value pairse that mimic early armada microservices.
- pattern:     Generate key/values based on user provided pattern, and then continually do CRUD operations on the keys.
- watch-tree:  Setup and monitor watches.

It can be used either as a standalone application, or run as a deployment in a Kubernetes cluster, deployed via a helm chart, that will run load against an
etcd-operator controlled etcd cluster. 

To run as a standalone application just build the code and execute etcd-driver.

To run as a deployment, first [check for and/or deploy image](./README.md#image-deployment), then check out [./imageDeploy](./imageDeploy/README.md) for details on deploying a test and processing results.

## etcd-driver pattern
`etcd-driver pattern ...` is the primary mode for using etcd-driver as it allows for simulation of an armada microservices etcd load.

### Pattern specification

The pattern generator, utilized by both the `pattern` and `armada` commands, relies on a pattern specification that has the following form:
```
pattern = <key-pattern>;<value-pattern>
key-pattern = /<level-pattern>[<key-pattern>]
value-pattern = <pattern>[<value-pattern>]
level-pattern = <default-rule-pattern>|<fast-value-pattern>|<regen-pattern>
pattern = <fast-value-pattern>|<regen-pattern>
default-rule-pattern = :[actual_desired|clusterid|masterid|region|state|workerid|ip] - Look at armadaPathRules in cmd/armada.go for further details
fast-value-pattern = #<size of value in bytes>
regen-pattern = <pattern that can be parsed by github.com/zach-klippenstein/goregen>, examples: [a-z0-9]{50}, clusters
```

Examples:
* /prefix/:ip/%level2-%06d[2]/%level3-%04d[6]/%level4-%04d[2]/%level5-%040d[100]/%level6-%040d[10]/%leaf7-%06d[5];[0-9]{140,160}
  This is the pattern used by the armada-etcd-siumlator helm chart (Q1 2021), and will create 120,000 keys.  
  key=/prefix/:ip/%level2-%06d[2]/%level3-%04d[6]/%level4-%04d[2]/%level5-%040d[100]/%level6-%040d[10]/%leaf7-%06d[5]  
    There are 7 levels to the key. 
    * `/prefix` - A hard code value
    * `/:ip` - Inserts the IP address of the container or host in which etcd-driver is running. This prevents etcd-drivers running in different pods from using the same set of keys. It also insures multiple etcd-drivers running in the same in the same pod, assuming they are using the same pattern, will use the same set of keys.
    * `/%level3-%04d[6]` - Creates 6 keys of the form `/level-####`.  
  value=[0-9]{140,160}: Creates a value of between 140 and 160 numeric characters.

### Control parameters

It is important to understand that the pattern defines levels in the key hierarchy, and that parameters to etcd-driver are used to control actions on the hierarchy by specifying the level where actions are to take place. Below is an overview of a subset of the parameters to `etcd-driver pattern`.

#### Key/value definition
```
--pattern string                       Pattern for keys/value pairs (required)
--val-spec string                      When set overrides the value regex. Comprabable to '[0-9]{n,m}' (default "0,0")
--total int                            Total number of key/value pairs to generate (set to 0 for when churning) (default 1).
```
#### Churn keys above a level
Churn-level performs create and delete operation on a segment of the key hiearchy. 
```
--churn-level int                      Key level to churn (0 based) (default 1)
--churn-level-pct int                  % of specified level to churn (default 10)
--churn-level-rate int                 Rate limit for deleting and putting keys at a level (updates/hour, 0 for unlimited) (default -1 which means disable churn-level)
```
#### Churn values
Writes new values on random keys at the specified rate. 
```
--churn-val-rate int                   Rate limit for updating values (puts/hour, 0 for unlimited) (default -1)
```
#### Gets
Does a range request on the specified level of the tree. It is quite possible that the churn-level functionality has deleted the keys being requested.
```
--get-level int                        Key level to churn (0 based) (default 1)
--get-rate int                         Rate limit for gets (gets/hour, 0 for unlimited) (default -1)
```
#### Watches
Setup watches on a portion of the key hierarchy, and optionally do a range get at a specified interval of all keys being watched. This mimics the armada rules engine load.
```
--watch-counts-per-level string        The number of watchers for each level, separated by ','. Level 0 should be the first digit, followed by level 1 etc. 'n' equates to all available keys at that level. Ex: '0,0,0,n,0,0'
--watch-prefix-get-interval duration   The duration between requests to get keys being watched
--watch-with-prefix                    Whether to specify 'WithPrefix' on the watch (match exact key or also sub-keys)
```
#### Output control
Result are written to `/churn_results.csv` in the pod. 
```
--csv-file string                      File to write csv results
--file-comment string                  Comment will be added to the 'file-comment' column of the csv output file. Useful for distinguishing runs stored in the same CSV file.
--stats-interval int                   The interval at which churn interval stats will be output (in seconds) (default -1)
```
The csv file has the following format. Note the meaning of the following prefix on each line:
* pattern-ramp - Stats for writing the key/value pairs specified by the pattern before churn is started.
* pattern-interval - Stats for the interval specified by `--stats-interval`
* pattern-summary - Stats for the entire test run. Excludes the pattern-ramp stats.
```
test,startTime,duration (ms),puts,puts/sec,deletes,deletes/sec,client deletes,client deletes/sec,gets,gets/sec,client gets,client gets/sec,key space,errors,Puts mean RT(μs),Puts min RT(μs),Puts   max RT(μs),Dels mean RT(μs),Dels min RT(μs),Dels max RT(μs),Gets mean RT(μs),Gets min RT(μs),Gets max RT(μs),Reconnects,Reconnect mean RT(μs),Reconnect min RT(μs),Reconnect max RT(μs),puts bytes/ sec,gets bytes/sec,watchers,watch events,watch events/sec,prefix gets,prefix gets/sec,client prefix gets,client prefix gets/sec,Prefix gets mean RT(μs),Prefix gets min RT(μs),Prefix gets max      RT(μs),prefix gets bytes/sec,churn-level,churn-level-pct,churn-level-rate,churn-val-rate,client-timeout,clients,conns,do-not-exit,endpoints,etcd-reconnect-count,file-comment,full-get-read,get-    keys-only,get-level,get-rate,pattern,put-rate,serializable-gets,skip-init,stats-interval,test-end-key,total,val-spec,verbose,watch-counts-per-level,watch-prefix-get-interval,watch-with-prefix
pattern-ramp,2020-11-13 21:20:20,105891,19241,181.7051,0,0.0000,0,0.0000,0,0.0000,0,0.0000,120000,100759,70759,14753,5975004,0,0,0,0,0,0,0,0,0,0,34073.95,0.00,0,0,0.0000,0,0.0000,0,0.0000,0,0,0,0.00,5,5,250,14000,10,20,20,true,1,0,,true,false,7,600000,"/prefix/:ip/%level2-%06d[2]/%level3-%04d[6]/%level4-%04d[2]/%level5-%040d[100]/%level6-%040d[10]/%leaf7-%06d[5];[0-9]{10,30}",0,false,     false,60,/test1/end,0,"10,30",false,,0s,false
pattern-interval,2020-11-13 21:21:20,60073,233,3.8786,57,0.9488,4,0.0666,1020,16.9793,2907,48.3910,120000,0,24127,7946,332033,15348,8969,25660,20619,8464,745969,0,0,0,0,69.45,3093.49,0,0,0.0000,0,0.0000,0,0.0000,0,0,0,0.00,5,5,250,14000,10,20,20,true,1,0,,true,false,7,600000,"/prefix/:ip/%level2-%06d[2]/%level3-%04d[6]/%level4-%04d[2]/%level5-%040d[100]/%level6-%040d[10]/%leaf7-%06d[5];[0-9]{10,30}",0,false,false,60,/test1/end,0,"10,30",false,,0s,false
.....
pattern-interval,2020-11-13 21:42:20,60000,234,3.9000,62,1.0333,4,0.0667,1967,32.7833,5500,91.6666,120000,0,8332,7366,47814,10783,4439,16159,10891,8222,59626,0,0,0,0,74.58,5977.38,0,0,0.0000,0,0. 0000,0,0.0000,0,0,0,0.00,5,5,250,14000,10,20,20,true,1,0,,true,false,7,600000,"/prefix/:ip/%level2-%06d[2]/%level3-%04d[6]/%level4-%04d[2]/%level5-%040d[100]/%level6-%040d[10]/%leaf7-%06d[5];[0-  9]{10,30}",0,false,false,60,/test1/end,0,"10,30",false,,0s,false
pattern-summary,2020-11-13 21:20:20,1382401,5328,3.8542,1669,1.2073,94,0.0680,44030,31.8504,123154,89.0870,120000,1,9170,6811,671398,7193,3917,25660,11108,7750,745969,0,0,0,0,722.35,5804.68,0,0,0.0000,0,0.0000,0,0.0000,0,0,0,0.00,5,5,250,14000,10,20,20,true,1,0,,true,false,7,600000,"/prefix/:ip/%level2-%06d[2]/%level3-%04d[6]/%level4-%04d[2]/%level5-%040d[100]/%level6-%040d[10]/%leaf7-    %06d[5];[0-9]{10,30}",0,false,false,60,/test1/end,0,"10,30",false,,0s,false
```
The csv file is organized as `test,startTime,duration (ms),<stat1>,...,<statN>,<parameter1>,...,<parameterM>`. If additional stats are added they must be added after existing stats on the line, and before the list of parameters. If this isn't done then the spreadsheet that depends on the order of columns will be broken. This order is defined by setupPatternStatKeys() and patPrintStats() in [cmd/user.go](./cmd/user.go).
#### Setup/Shutdown
```
--put-rate int                         Number of keys to put per hour during loading of the initial keys (0 is no limit)
--skip-init                            Skip the loading of the initial keys and starts churn. Useful if two instances of etcd-driver are operating on the same key hierarchy, which is the case for the armada-etcd-simulator helm chart.
--do-not-exit                          Don't exit the program after final statistics are published. This should be used when deploying to a container. Otherwise when signaling that the test is done, etcd-driver will exit causing the pod to be restarted and thus the load to be restarted.
--test-end-key string                  A key that will be watched, and when set to true the watchers will terminate (default "/prefix/testEnd"). When the key writen etcd-driver will write the `pattern-summary` st 
```
#### Connections and threads
```
--client-timeout int                   The timeout (in seconds) used for etcd interactions (default 10 seconds) (default 10)
--clients uint                         Total number of gRPC clients (default 1). This translates to the total number of GO threads, that will use the configured connections (i.e. --conns) to deliver load.
--conns uint                           Total number of gRPC connections that will be initiated between the client and etcd (default 1)
```

## Scripts

This is a set of scripts, created in 2017/8, that was used to drive load via a single instance of `etcd-driver pattern` and a single instance of `etcd-driver watch-tree`. There are references to tests against both kubernetes and armada microservice etcd. Some scripts (ex: `run_all_churn_tests.sh`) are hardcoded to use the `etcd-slnfs` database, which would have been created by the scripts in [../scripts](../scripts) repo. Basically an early attempt to examine etcd load charecteristics. See [Etcd realistic workload notes](https://ibm.ent.box.com/notes/138662981774).
* `all24HourTests.sh` - 
* `endTest.sh` - 
* `etcdCompressDb.sh` - 
* `etcdDumpStats.sh` - 
* `getDBsize.sh` - 
* `persistent-claim.yml` - 
* `runWatcher.sh` - 
* `runChurn.sh` - 
* `run_churn_test.sh` - 
* `run_all_churn_tests.sh` - 


## Image deployment
### Check to see if the etcd-driver benchmark image is already in the registry
Log in to IBM Cloud
```
ibmcloud login -a  https://test.cloud.ibm.com (dev/stage)
ibmcloud login -a  https://us-south.containers.cloud.ibm.com (production)
```
Log in to the appropriate registry
```
ibmcloud cr api https://stg.icr.io/api (dev/stage)
ibmcloud cr api https://us.icr.io/api (production)
```
View the current images in the registry:
```
ibmcloud cr images
```
Check that the armada_performance namespace exists (it probably does)
```
ibmcloud cr namespaces
```
Create the armada_performance namespace if it is missing
```
ibmcloud cr namespace-add armada_performance
```
If the required etcd-driver benchmark image is already available then you can skip the next section on 'Building and Uploading the etcd-driver benchmark Image to the Registry'.

### Building and Uploading the etcd-driver benchmark Image to the Registry

If the desired etcd-driver benchmark image is not already in the registry, build the docker image and upload it to the registry.

These instructions assume you have cloned the armada-performance repo on your client machine.

On the client machine (replace <GITHUB_ROOT> with the location of the alchemy-containers organisation)

Build the etcd-driver application locally:
```
cd <GITHUB_ROOT>/alchemy-containers/armada-performance
glide install
cd <GITHUB_ROOT>/alchemy-containers/armada-performance/etcd/etcd-driver
CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
cd <GITHUB_ROOT>/alchemy-containers/armada-performance
```

Build the docker image locally. (Examine the Dockerfile to see how it builds the image). The image MUST be built from the above directory
```
docker build -t etcd-driver -f etcd/etcd-driver/imageCreate/etcd-driver/Dockerfile .
```
Tag the image to point at the appropriate repository and namespace (below uses stage registry, with the [optional] stage1 namespace and a image version of latest as an example)
```
docker tag etcd-driver stg.icr.io/armada_performance[_stage1]/etcd-driver:latest (dev/stage)
```

Login and Push the image to the IBM Cloud Registry (example use stage registry)
```
docker login -u iamapikey -p <your_apikey> stg.icr.io
docker push stg.icr.io/armada_performance[_stage1]/etcd-driver:latest
```
(Use `STAGE_GLOBAL_ARMPERF_IBMCLOUD_APIKEY` from Vault or your own APIKEY

View the image in the registry
```
ibmcloud cr images
```

