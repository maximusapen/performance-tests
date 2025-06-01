# Cruiser Churn Tool
Cruiser_churn creates, updates the kube version and then deletes cruisers. It effectively accelerates the change that would be seen in production in order to introduce variability into our test environments. It attempts to maintain an approximate number of cruisers on the carrier, whilst continuously creating, updating and deleting cruisers.

The approach taken is as follow:
1. Collect a snapshot of existing cruisers in the carrier
2. Randomly select a cruiser to change. It could be a cruiser that doesn't exist if the total found is less than the number of cruisers requested by the `-clusters` flag
3. Initiate a create if cruiser doesn't exist, update the kube version if the current version is not the version defined by `kubeVersion` or delete if the kube version has already been upgraded.
4. Collect and publish metrics on the operation
5. When a delete operation completes then a create is initiated so that the number of cruisers stays close to the maximum requested.

# Example run
Many of the parameters to cruiser_churn are the same as armada-perf-client (ex: -machineType) and are necessary to define the cruisers to be created. Below is an example of using cruiser_churn and a description of the parameters that impact characteristics of the churn.

`./cruiser_churn -action ChurnClusters -clusterNamePrefix fakecruiser-base- -clusters 1300 -workers -1 -machineType u2c.2x4 -numThreads 20 -monitor -metrics -verbose=false -masterPollInterval    30s -kubeVersion 1.15`

* `-clusterNamePrefix <cruiser-name-prefix>` - Defines, possibly a subset, of cruisers that will be churned.
* `-clusters <# clusters>` - The number of cruisers in the set to be churned.
* `-numThreads <threads>` - Defines that maximum number of CRUD operations that will be run simultaneously.
* `-kubeVersion 1.15` - The kube version used in upgrades. Newly created cruisers get the default kube version for carrier.

The current values used for number of threads and number of cruisers to maintain on carrier5 can be found in https://github.ibm.com/alchemy-containers/armada-performance/blob/master/api/cruiser_churn/carrier5.env

`FAKE_` denotes cruisers that have 0 workers

`REAL_` denotes cruisers that have 1 worker

# Setup
Assumes performance code has been deployed under `stgiks-dal10-perf##client-##:/performance`

Cruiser churn will automatically run against the carrier number that matches the perf-client.

You can either use the following job -> https://alchemy-containers-jenkins.swg-devops.com/job/Armada-Performance/job/perfClient-setup/job/install-multiple-prereqs-on-perf-client/ and select `install-cruiserchurn.yaml` or install manually using the following instructions:

```
mkdir -p /performance/stats/churn
sudo cp /performance/armada-perf/api/cruiser_churn/*churn.service /lib/systemd/system
sudo cp /performance/armada-perf/api/cruiser_churn/*churn.service /etc/systemd/system
sudo chmod 644 /etc/systemd/system/*churn.service
```

To control cruiserchurn, you can either use this job -> https://alchemy-testing-jenkins.swg-devops.com/job/Armada-Performance/job/Utils/job/Control-CruiserMon-CruiserChurn/ or use the commands below directly on the perf-client.

```
sudo systemctl start fakecruiserchurn
sudo systemctl enable fakecruiserchurn
sudo systemctl start realcruiserchurn
sudo systemctl enable realcruiserchurn
```
If you also want to run churn of openshift cruisers then use the following:
```
sudo systemctl start fakeopenshiftchurn
sudo systemctl enable fakeopenshiftchurn
sudo systemctl start realopenshiftchurn
sudo systemctl enable realopenshiftchurn
```
Check cruiser churn is running using:
```
sudo systemctl status fakecruiserchurn
sudo systemctl status realcruiserchurn
```
For openshift churn:
```
sudo systemctl status fakeopenshiftchurn
sudo systemctl status realopenshiftchurn
```

Once running and enabled it will automatically restart if the process unexpectedly terminates or the host reboots.

The logs for all churn will be written to `/performance/stats/churn/cruiser_churn.log`

Results can be monitored at [Grafana](https://alchemy-prod.hursley.ibm.com/stage/performance/grafana/d/N6sEBkMWk/_cruiserchurn) (Select the Carrier you are interested in in the drop-down)


# Shutdown
Stop cruiser churn using:
```
sudo systemctl stop fakecruiserchurn
sudo systemctl stop realcruiserchurn
```
For Openshift churn use:
```
sudo systemctl stop fakeopenshiftchurn
sudo systemctl stop realopenshiftchurn
```
Note - these commands can take up to 1 hour (usually less than 30 mins) to return - as they wait for any in-flight creates/updates/deletes to finish before exiting.
(Note it will be started again on reboot unless you run):
```
sudo systemctl disable fakecruiserchurn
sudo systemctl disable realcruiserchurn
```
For Openshift churn use:
```
sudo systemctl disable fakeopenshiftchurn
sudo systemctl disable realopenshif
