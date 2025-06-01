# etcd-driver/imageDeploy

## Running as a deployment against an etcd-operator cluster

There are two helm charts for deploying etcd-driver: 
* etcd-driver - Deploys one or more pods each running a single instance of `etcd-driver pattern ...`
* armada-etcd-simulator - Deploys one or more pods running two instances of `etcd-driver pattern ...`. One instance drives CRUD on the key/value pairs, the other sets up and monitors watches.

The charts are controlled via a set of variables defined in `<chart name>/values.yaml`. The following values control the overal layout/functionality of the deployment:
* secretsPrefix - If set to '' then encryption won't be enabled. If secretsPrefix is set to anything other than '' then the following files must be available in the {{ .Values.secretsPrefix }}-client-tls secret:
  * etcd-client.crt
  * etcd-client.key
  * etcd-client-ca.crt
* parameters.endpoints - If defined then the specified endpoints will be used to connect to etcd. Otherwise an endpoint will be constructed based on other values: `https://{{ .Values.prefix }}-client.{{ .Values.namespace }}.svc:2379` (ex: `https://etcd-5node-client-armada.svc:2379`)
* parameters.watchLevelCounts - Must be set for the etcd-driver container driving the watch load to be created. Only applies to armada-etcd-simulator.
* parameters.leaseStartLeases - Must be set for the lease_test container driving the lease load to be created. Only applies to armada-etcd-simulator.


## Running tests
The following scripts are used for managing test runs:
* `deploy_etcd_driver.sh <etcd cluster name prefix> <namespace> [<helm chart name, defaults to 'etcd-driver'>]`
* `delete_etcd_driver.sh <etcd cluster name prefix> <namespace>|all [<helm chart, defaults to 'etcd-driver'>]`
* `stop_test_collect_results.sh <etcd cluster name> <namespace> [<true>=>skip etcd-driver test end trigger]` - Stops etcd-driver test, collect results including etcd-driver and etcd logs, and summarizes results. 

The process for running tests are as follows:
1. Setup an etcd cluster. See [test-harness](../../test-harness/README.md).
1. `cp ../../test-harness/etcd-perftest-config .`
1. `./deploy_etcd_driver.sh etcd-5node armada armada-etcd-simulator`
1. The default deploy can take 30-45 minutes to setup, so wait at least 2 hours before stopping the test
1. `./stop_test_collect_results.sh etcd-5node armada armada-etcd-simulator`
1. Populate the spreadsheet with the resulting `churn_results.csv`
1. Examine the output of error processing.  
1. `./delete_etcd_driver.sh etcd-5node armada armada-etcd-simulator`  

The etcd-driver pods remain until deleted, though they won't generate load after test is stopped. The exception is if nodes are rebooted, in which case they will start generating load.

The etcd-driver will start sending etcd requests as soon as the pod starts. To monitor throughput and results you can either look at the pod logs:
```
kubectl logs -n <namespace> <podName>
```
or copy the `churn_results.csv` file from the pod:
```
kubectl cp <namespace>/<podName>:/churn_results.csv ./churn_results.csv
```
e.g.
```
kubectl cp etcd-operator-1/testetcd-eo1-100-th9-9-etcd-etcd-driver-5b8f67f854-qz8mr:/churn_results.csv ./churn_results.csv
```

## Processing the results of a test run
1. Populate the `armada-etcd-simulator-results.xlsx` spreadsheet with the resulting `churn_results.csv`
1. Examine the output of the stop script.

  The following example of stopping the tests stores etcd-driver logs in `./backup/logs.<etcd-driver pod name>.<container name>.2021-01-21-15-30.log`, etcd logs in `./backup/<etcd container name>.2021-01-21-15-34.log, etcd-driver log analysis in `./backup/2021-01-21-15-30.test.summary.txt`, and etcd log analysis in `./2021-01-21-15-34.etcd.test.summary.txt`. The `churn_results.csv` file is moved to `churn_results.2021-01-21-15-30.csv`.
  ```
  $ ./stop_test_collect_results.sh etcd-5node armada armada-etcd-simulator
  Endpoints: 52.117.182.101:32278
  Thu Jan 21 15:30:41 EST 2021
  OK
  OK
  Stop: log label: 2021-01-21-15-30
  Thu Jan 21 15:31:42 EST 2021
  tar: removing leading '/' from member names
  ....
  tar: removing leading '/' from member names
       100 pod/containers reported summary statistics
  Etcd log label: 2021-01-21-15-34
  Error data: backup/2021-01-21-15-30.test.summary.txt
  Etcd log analysis parameters: 2021-01-21-15-34 2021-01-21 19:4* 2021-01-21 20:30
  Processing: backup/etcd-5node-6vghqn97nx.2021-01-21-15-34.log
  Processing: backup/etcd-5node-74s5t4krl9.2021-01-21-15-34.log
  Processing: backup/etcd-5node-d8gl6245tz.2021-01-21-15-34.log
  Processing: backup/etcd-5node-tcrf9slsbb.2021-01-21-15-34.log
  Processing: backup/etcd-5node-xb7t98wfjh.2021-01-21-15-34.log
  Etcd error data: backup/2021-01-21-15-34.etcd.test.summary.txt
  ```
## Utilities/sub scripts
* `armada_get_etcd_logs.sh` - Gets the armada etcd logs (etcd_cluster=etcd-501-armada-stage5-south, namespace=armada)
* `get_etcd_connections.sh  [<etcd_cluster> [<namespace>]]` - Extracts etcd container connections. 
* `get_etcd_logs.sh [<etcd cluster> [<namespace>]]` - Extracts etcd member logs and puts results in backup/<container>.$(date +"%Y-%m-%d-%H-%M").log. Called by `stop_test_collect_results.sh`.
* `restartEtcdClusterPods.sh` - Slowly restarts etcd pods in cluster. Primarily used to mimic nodes being rebooted.
* `summarize_errors.sh <log timestamp>` - Summarizes the errors in the etcd-driver found in backup/\*.<log timestamp>.log. Called by `stop_test_collect_results.sh`.
* `summarize_etcd_errors.sh <log timestamp> <start time prefix> <end time prefix>` - Summarizes the erros in the etcd cluster logs found in backup/\*.<log timestamp>.log. Called by `stop_test_collect_results.sh`.


Include?
* `check_pods_running.sh <prefix> <namespace> [helm charg]` - 

## Deploy and run the etcd-driver benchmark on a Kubernetes Cluster

The following Jenkins jobs can be used to deploy and delete the etcd-driver application. The job is somewhat out of date, as it doesn't support the optional `[<helm chart name, defaults to 'etcd-driver'>]` parameter to `deploy_etcd_driver.sh`. As a result it can only be used to deploy the default chart (i.e. etcd-driver).

* https://alchemy-testing-jenkins.swg-devops.com/job/Armada-Performance/job/Utils/job/DeployEtcdDriver/
* https://alchemy-testing-jenkins.swg-devops.com/job/Armada-Performance/job/Utils/job/DeleteEtcdDriver/
