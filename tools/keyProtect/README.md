# Key Protect

## Enable Key Protect

Instructions to enable Key Protect in https://test.cloud.ibm.com/docs/containers?topic=containers-encryption#keyprotect 
(Note:  prod is https://cloud.ibm.com/docs/containers?topic=containers-encryption#keyprotect).

Steps below are based on the docs above.  Instructions may change so always check with doc.

### Login to ibmcloud targeting the region for the test

We ran tests on stage in us-south

    ibmcloud login -a https://test.cloud.ibm.com -r us-south -g default -u armada.performance@uk.ibm.com -p < password >

We ran tests on prod in jp-tok in order to gather Key Protect metrics

    ibmcloud login -a https://cloud.ibm.com -r jp-tok -g default -u armada.performance@uk.ibm.com -p < password >

### Key Protect service instance

Instructions in https://test.cloud.ibm.com/docs/services/key-protect?topic=key-protect-provision

with

- resource_group_name: default

Stage:

- region_name: us-south

Prod:

- region_name: jp-tok

Main steps:

- Create kp service instance

    ibmcloud resource service-instance-create <kp_instance_name> kms tiered-pricing us-south

- Get instance_ID

    ibmcloud resource service-instance <kp_instance_name>  | grep GUID

Example output of "ibmcloud resource service-instance <kp_instance_name>":

```
Name:         perf-kp
ID:           crn:v1:staging:public:kms:us-south:a/4a160c3a25d49f6171b796555191f7da:5cd15368-be0f-47af-bdb6-9967befccafb::
GUID:         5cd15368-be0f-47af-bdb6-9967befccafb
Location:     us-south
State:        active
Type:         service_instance
Sub Type:     kms
Created at:   2019-07-05T15:49:43Z
Updated at:   2019-07-05T15:49:43Z
```

and instance_ID is 5cd15368-be0f-47af-bdb6-9967befccafb

- List resources

List all kms instances:
    ibmcloud ks kms instance ls

List the root key for a kms instance using instance id (not instance name):
    ibmcloud ks kms crk ls --instance < kms instance id >


- Delete kp service instance

    ibmcloud resource service-instance-delete <kp_instance_name>

### Customer Root Key (crk)

Instructions in https://test.cloud.ibm.com/docs/services/key-protect?topic=key-protect-create-root-keys#create-root-keys

kms_endpoint:

- stage:  qa.us-south.kms.test.cloud.ibm.com
- prod:   jp-tok.kms.cloud.ibm.com

Main steps:

- Get IAM token:
    ibmcloud iam oauth-tokens

- Create crk with the IAM token and instance_ID with curl call:

```
 curl -X POST \
   https://<kms_endpoint>/api/v2/keys \
   -H 'authorization: Bearer <IAM_token>' \
   -H 'bluemix-instance: <instance_ID>' \
   -H 'content-type: application/vnd.ibm.kms.key+json' \
   -d '{
  "metadata": {
    "collectionType": "application/vnd.ibm.kms.key+json",
    "collectionTotal": 1
  },
  "resources": [
    {
    "type": "application/vnd.ibm.kms.key+json",
    "name": "<key_alias>",
    "description": "<key_description>",
    "extractable": false
    }
  ]
 }'
 ```

Example output of curl call when crk created successfully with crk_ID:

```
{"metadata":{"collectionType":"application/vnd.ibm.kms.key+json","collectionTotal":1},"resources":[{"id":"<crk_ID>","type":"application/vnd.ibm.kms.key+json","name":"perf-kp-rootkey","state":1,"crn":"crn:v1:staging:public:kms:us-south:a/4a160c3a25d49f6171b796555191f7da:<instance_ID>:key:<crk_ID>","extractable":false,"imported":false}]}
```

### List resources

List all kms instances:
    ibmcloud ks kms instance ls

List the root key for a kms instance using instance id (not instance name):
    ibmcloud ks kms crk ls --instance < kms instance id >

### Delete resources

First, delete crk with the IAM token and instance_ID with curl call:

```
curl -X DELETE \
   https://qa.us-south.kms.test.cloud.ibm.com/api/v2/keys/<crk_ID> \
   -H 'authorization: Bearer <IAM_token>' \
   -H 'bluemix-instance: <instance_ID>' \
   -H 'prefer: representation'
```

Then delete the kms instance with:
    ibmcloud resource service-instance-delete < kms instance name or id >


### Enable Key Protect in a cluster

Enable with command:

    ibmcloud ks kms enable -c <cluster_name_or_ID> --instance-id <kms_instance_ID> --crk <root_key_ID>

Then run

    ibmcloud ks cluster get --cluster <cluster_name>

`Master Status` will show key protect is being requested/processed.  It normally takes 20 minutes for master to show `Ready` state.

On the carrier, you can see that master pods for the cluster has an additional kms container after cluster is key protect enabled, i.e. a total of 5 containers instead of 4.

### Secret

Scripts in keyProtect/secret can be used to add/delete many secrets to one or many clusters for testing.

To generate secret yaml, modify keyprotect/secret.yaml with real data first.  For example, copy data from output of

    kubectl get secret default-stg-icr-io -o yaml

### KMS config patching

Scripts in tools/keyProtect/config are used to make configuration changes to kms for all test clusters, to be run on carrier master.

- patch_cache_timeout.sh: Modify CACHE_TIMEOUT_IN_HOURS from default of 1 to 0
- patch_cachesize.sh: Modify cachesize in cruiser master configmap
- scalemaster.sh: Scale all master pods to 0 for emergency use.

All cruisers will restart master pod as soon as any one of the patch*.sh is run.  Once patched, all DEKS will collapse to 1 DEK.
Make changes to the scripts to change all config back to default after/during test.

## Tests

Test scripts provided should work for all tests.  There are 3 test scripts in tools/keyprotect directory

- test_conf.sh:  Configure common parameters for scripts in config and secret directory.  Modify for your test.
- test_clusters.sh:  Test many clusters.
- test_one_master.sh.  Test one cluster

Also modify the following parameters in the script for your test if declared in script:

secretStart=1
secretEnd=1
declare -i clusterStart=1
declare -i clusterEnd=950

### 80 clusters test

Tests are run in 2 environments:

- Client (your Mac or perf client) to run ibmcloud ks commands
- Carrier master to patch cruiser masters for restarting masters
  - "ibmcloud ks cluster refresh" command does not provide enough stress for this test as the master restart runs in batches

Test steps after logging into IKS performance carrier:

1. Create 80 1-worker clusters with no sub-net.  Name of clusters are kpcluster1 to kpcluster80.
    - create_clusters_1.sh (Modify kubeVersion if dev team provides test BOM)
   Alternatively, run this command on perf client:
    - nohup /performance/bin/armada-perf-client -action AlignClusters -clusterNamePrefix kpcluster -clusters 80 -workers -1 -machineType u2c.2x4 -numThreads 25 -workerPollInterval 30s -masterPollInterval 30s -kubeVersion <kube-version> -monitor -useChurnVLAN &
2. Get cluster config for all clusters:
    - get_cluster_configs_2.sh
3. Enable Key Protect for all clusters after creating Key Protect service instance and Customer Root Key as described above:
    - enable_kp_3.sh (Update kpInstance and crk)
4. Generate 500 secrets data
    - generate_secrets_4.sh (Modify data in secret.yaml - see Secret section above)
5. Add 200 secrets to all clusters
    - add_secrets_5.sh 1 200
6. Now run tests in parallel
    - Client: Continue to add secrets to clusters:
      - add_secrets_5.sh 201 300
      - add_secrets_5.sh 301 400
      - etc.... while test is still ongoing
    - Carrier master: Restart all clusters master pods by patching kms container in round robin:
      - patch_cache_timeout.sh 0 (Setting CACHE_TIMEOUT_IN_HOURS: 0 is likely to cause master pods to go in CrashLoopBackOff state)
      - patch_cache_timeout.sh 1 (Setting CACHE_TIMEOUT_IN_HOURS: 1 is likely to have no issue and hopefully can cause master pods to recover from CrashLoopBackOff)

Monitor Step 6 while adding secrets and patching masters for any master pods running into CrashLoopBackOff state.  Once you see CrashLoopBackOff state, monitor to see whether it will recover itself with no manual intervention.

Manual Recovery of masters pods in CrashLoopBackOff:

- If all master pods goes into CrashLoopBackOff and won't recover, then scale all master pods to 0:
  - scale_masters.sh 0
- Before scaling back to 3 replicas, if "CACHE_TIMEOUT_IN_HOURS: 0" has been set, then you want o set it back to default 1:
  - patch_cache_timeout.sh 1
- Scale back to 3 replicas:
  - scale_masters.sh 3

Test cleanup:

- Delete all test secrets from all 80 clusters, say 400 secrets from all 80 clusters:
  - delete_secrets_6.sh 1 80 1 400
- Delete 80 clusters
  - delete_clusters_7.sh
  Alternatively, run this command on perf client:
  - nohup /performance/bin/armada-perf-client -action AlignClusters -clusterNamePrefix kpcluster -clusters 0 -workers -1 -machineType u2c.2x4 -numThreads 25 -workerPollInterval 30s -masterPollInterval 30s -kubeVersion <kube-version> -monitor -useChurnVLAN &
- Delete the root key from KP endpoint (see above)
- Delete the kms service instance (see above)

### 950 clusters with 20 DEK tests

Same test environment as in the 80 clusters tests above with below test steps:

1. Create 950 0-worker clusters with no sub-net.  Name of clusters are kpcluster1 to kpcluster950.
    - create_clusters_1.sh (Modify kubeVersion if dev team provides test BOM)
2. Get cluster config for all clusters:
    - get_cluster_configs_2.sh
3. On carrier master run script to monitor key protect enablement.
    - nohup monitor_master_restart.sh --enablekp &
4. Enable Key Protect for all clusters after creating Key Protect service instance and Customer Root Key as described above:
    - enable_kp_3.sh (Update kpInstance and crk)
5. Generate 20 secrets data for 950 clusters
    - generate_secrets_4.sh (Modify data in secret.yaml - see Secret section above)
6. On carrier master run script to monitor master pod status.  Script will run forever.  Kill process after test.
   Alternatively Use a new process for every loop run in step (7) so you can save the nohup.out for each DEK increase
    - nohup monitor_master_restart.sh &
7. Run in loop to increase number of DEK from 1 to 20
    - On carrier master run script to monitor master pod status.  Script will run forever.  Kill process after each loop run.
        - nohup monitor_master_restart.sh &
    - Add 1 secret to all clusters to increase DEK by 1
        - add_secrets_5.sh (Start secretStart and secretEnd with 1 and increasing by 1 for each loop)
    - Get all secrets for all clusters
        - get_secrets_6.sh
    - On carrier master, restart all masters, i.e. 950 clusters X 3 masters = 2850 masters
        - restart_masters.sh (Script will restart all 5 container master pods on carrier - should take about half an hour)
    - Check nohup.out from monitor_master_restart.sh.  If all masters are restarted with no master pods activities
        - Kill the monitor_master_restart.sh process
        - mv nohup.out dek<number of DEK>.log
8. Don't forget to kill the monitor_master_restart.sh process.

### Automated 950 clusters with multiple DEK tests

Tests are now automated with scripts:

- test_conf.sh
- test_clusters.sh
- test_one_master.sh

Modify test_conf.sh for test if necessary.

The armada-master-resource-monitor pod on the carrier restarts master pods on different nodes
based on resource usage.  This affects test result which counts number of Runningmaster pods
and the number of master pod restarts.  Before starting test, scale the armada-master-resource-monitor
pod on the carrier to 0:
    kubectl scale deployment -n armada armada-master-resource-monitor --replicas 0

Make sure armada-master-resource-monitor is scaled back to 1 after test.

Run with command with console output to nohup.out
    nohup ./test_clusters.sh &

Useful data from nohup.out for reporting with example results

- grep DEKs nohup.out
  - Restart time 109 DEKs:  349 masters restarted more than once with CrashLoopBackOff
  - Restart time 109 DEKs:  13 minutes and 38 seconds
  - Restart time 110 DEKs:  545 masters restarted more than once with CrashLoopBackOff
  - Restart time 110 DEKs:  13 minutes and 52 seconds
    - Use "| grep CrashLoopBackOff" and "| grep -v CrashLoopBackOff" to filter and build Excel graphs.
- grep Secret nohup.out
  - Restart time Secret 1 : 732 sec
  - Restart time Secret 2 : 623 sec
  - Restart time Secret 3 : 620 sec
    - Use result to build Excel graphs.
- grep "Time to" nohup.out
  - 107: Time to scale up all clusters: 8 minutes and 49 seconds.
  - Time to add secret(s) for all clusters: 3 minutes and 1 seconds (181 seconds).
  - Time to get all secrets for all clusters: 48 minutes and 37 seconds (2917 seconds).
  - 108: Time to scale up all clusters: 8 minutes and 49 seconds.
  - Time to add secret(s) for all clusters: 3 minutes and 1 seconds (181 seconds).
  - Time to get all secrets for all clusters: 20 minutes and 34 seconds (1234 seconds).
- grep "Time to get all secrets" nohup.out
  - Use result to build Excel graphs.

## Tests

Test reports in Box folder https://ibm.ent.box.com/folder/92209268437:

- Small 5-workers cluster: https://ibm.ent.box.com/notes/484368900989 
- Large 100-workers cluster: https://ibm.ent.box.com/notes/491888754946
- 80 clusters with 1-worker: https://ibm.ent.box.com/notes/495810771736
- DEK cache: https://ibm.ent.box.com/notes/498664085569
- Small cluster with 3000 secrets in JP-Tok with KP metrics: https://ibm.ent.box.com/notes/506138228361
- Enable Key Protect on Tugboat: https://ibm.ent.box.com/notes/551474804314
- Key Protect scale testing on large carrier: https://ibm.ent.box.com/notes/552188853812
- Impact of Key Protect masters restart with increased DEKs: https://ibm.ent.box.com/notes/560239176799
- IKS KP Capacity & Rate Limit Testing: https://ibm.ent.box.com/notes/570920073540
- Stress load testing with KP squad (950 clusters with high number of DEKs - stretch goal of 120):
  - https://ibm.ent.box.com/notes/570920073540
  - https://ibm.ent.box.com/notes/574699676272?s=995t0snlms6vlt4wpzlvfx9emeuorq9i
  - https://ibm.ent.box.com/notes/619292212925

Performance Tests covered:

- K8s-E2e-Performance-Load test for small KP cluster
- APIServer-Load test for 80 1-worker KP cluster
- Manual K8s-E2e-Performance-Load test for big 100-worker cluster in https://github.ibm.com/alchemy-containers/armada-performance/blob/master/scripts/run_manual_e2e.sh - with requests.csv generated by keyprotect/secret/generate_requests_csv.sh
