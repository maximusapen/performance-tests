# Test armada-api with Jmeter

Test armada-api of a carrier/tugboat with [Jmeter](https://github.ibm.com/alchemy-containers/armada-performance/tree/master/jmeter-dist)
Example here is using stage carrier4.

The scripts in this directory supports the test `armada-api-load` in runAuto.sh.  You can also run the test manually on
a performance client.  See steps below.

## Jmeter stress load

The requests.csv file is requesting the workers of a cluster.  So, you need to create a cluster and modify the requests.csv with the cluster id.

For classic:

- GET,/v2/classic/getWorkers?cluster=<cluster_id>

For vpc-gen2:

- GET,/v2/vpc/getWorkers?cluster=<cluster_id>

Below are some other example of requests you can put in requests.csv which doesn't require a cluster"

- GET,/v2/classic/getClusters
- GET,/v2/vpc/getClusters
- GET,/v2/getVersions

You can have a mixture of requests in requests.csv.

Run test-armada-api.sh in nohup with these positional parameter:

- nohup ./test-armada-api.sh <carrier> <interval in sec> <thread number> <thread req limit in min> [<summary result file>] &

    The <summary result file> is optional.  If not passed in, summary will be in summary.out file.

Examples:
  Run Jmeter for 10 min with 5 threads limiting each thread to 1200 req/min (20 req/sec) and write summary to default summary.out:
    test-armada-api.sh origin-4 600 5 1200
  Run Jmeter for 20 min with 2000 threads limiting each thread to 1200 req/min (20 req/sec) and write summary to summary_1000.out:
    test-armada-api.sh origin-4 1200 2000 1200 summary_1000.out

Warning: Do not set interval for longer than 1 hour or you will get RC-401,Unauthorized.

You can monitor the test run with:

- tail -f nohup.out

You can check the total number of RC-500, RC-503, RC-401... errors in results.jtl with command:

- checkRC.sh <results jtl file>

## Cruiser churn load

For manual runs, you may need extra load on the carrier with cruiser churn in addition to the Jmeter stress load.

For `stage-dal10-carrier4` create a new file called `/performance/armada-perf/api/cruiser_churn/carrier4.env` with following contents:

FAKE_CLUSTERS=200
FAKE_THREADS=10
REAL_CLUSTERS=100
REAL_THREADS=5
FAKE_OPENSHIFT_CLUSTERS=440
FAKE_OPENSHIFT_THREADS=10
REAL_OPENSHIFT_CLUSTERS=10
REAL_OPENSHIFT_THREADS=3

Carrier `stage-dal10-carrier4` has spoke tugboat `stage-dal10-carrier500` hosting the cruiser master, edit `/performance/armada-perf/api/cruiser_churn/startChurn.sh` and ensure KUBECONFIG is pointing at `stage-dal10-carrier500`:

export KUBECONFIG=/performance/config/carrier500_stage/admin-kubeconfig

To start cruiser churn:

- sudo systemctl start realcruiserchurn

To stop cruiser churn:

- sudo systemctl stop realcruiserchurn

## Rollout

Script rollout.sh is included which was used when testing `kubectl rollout restart` to compare with `armada-secure` rollout.  The `armada-secure` rollout should be implementing logic very similar to `kubectl rollout restart`.  The script allows us to compare Jmeter errors in both rollouts.

### Reference

Testings using the tools here:

- [Performance impact of secure rollout vs kubectl rollout](https://ibm.ent.box.com/notes/854719175274)
- [Performance impact of secure rollouts - Test 1](https://ibm.ent.box.com/notes/847155575852)
- [Performance impact of secure rollouts - Test 2 & Test 3](https://ibm.ent.box.com/notes/848485306358)
 -[Improvements in armada-api - batch gets & authentication caching](https://ibm.ent.box.com/notes/863523468712)
