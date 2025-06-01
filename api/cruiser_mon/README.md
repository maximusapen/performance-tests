# Cruiser Monitoring Tool
Cruiser_mon monitors cruisers in a carrier and generates metrics on their accessibility. It regularly checks kubx-masters configmaps to determine what cruisers have been added or deleted and updates the set of cruisers it is monitoring. Each monitoring cycle it tests whether the cruiser's kube apiserver port is active and then requests a list of services from kube. It reports the following metrics to the metrics service:

* Number of successful request for kube services
* Number of failed attempts
* Number of soft failures (i.e. a second attempt was successful)
* Number of cruisers that are booting
* Number of cruiser where booting failed
* Number of cruisers that are being deleted
* Minimum, maximum and average response times to the request for services.

A single cruiser will only be counted in one of the above categories.

## Logging
Cruiser_mon publishes data to `/performance/stats/cruiser_mon/nohup.out` every polling cycle. If all cruisers are responsive then the output will be a single line summary for the cycle with each field matching the metrics mentioned above:
```
2019-02-01 15:56:49, 1263, 0, 0, 0, 0, 0, 8.135789ms, 309.94087ms, 28.772288ms
```

Each cycle when a cruiser isn't successfully polled then Mean Time To Repair (MTTR) data is published to indicate the nature of the failure (`MTTR_DATA`) and the results once the failure resolves (`MTTR_RESULT`).

```
MTTR_DATA: 2019-01-17 16:43:50, 02c664b0f04a4bf0a95d8c46febc886e, true, true, false false
```

* Date & time
* Cruiser ID
* Is cruiser is in `running` state?
* Is cruiser port active?
* Was kube `get svc` call successful?
* Did a soft failure occur?

The following is an example of the possible MTTR_DATA messages
```
# Minimum of one master pods isn't running
MTTR_DATA: 2020-04-28 14:40:35, bqjml1c20qgp6fmgapq0, false, false, false false
# Port dial unsuccessfully, true twice
MTTR_DATA: 2020-04-28 14:40:35, bqjml1c20qgp6fmgapq0, true, false, false false
# Get services request failed, tried twice
MTTR_DATA: 2020-04-28 14:40:35, bqjml1c20qgp6fmgapq0, true, true, false false
# Soft failure: Second of two get services requests succeeded.
MTTR_DATA: 2020-04-28 14:40:35, bqjml1c20qgp6fmgapq0, true, true, true true
# Soft port dial failure, get services requests failed. This state isn't reported in the logs.
MTTR_DATA: 2020-04-28 14:40:35, bqjml1c20qgp6fmgapq0, true, true, false true
# End of outage and outage stats
MTTR_DATA: 2020-04-28 14:43:02, bqjml1c20qgp6fmgapq0, true, true, true
```


```
MTTR_RESULT: 2019-01-17 16:48:37, 025e3d7179b748b589b3b457335d47dd, 2019-01-17 16:43:51, 4m45.435659292s
```

* Date & time
* Cruiser ID
* Date & time of the start of the outage
* Duration of the outage

# Example run
```
./cruiser_mon -prefix  --timeout 30s --loop 20s -configmaps -dir ./carrier5-nfs -carrier carrier5
```

* `--timeout 30s` - Timeout for kube get svc call
* `--loop 20s` - The polling period
* `-configmaps` - Monitor kubx-masters configmaps to find cruisers
* `-dir ./carrier5-nfs` - Directory where kube cluster configurations are stored
* `-carrier carrier5` - The carrier to be monitored. This is added to metric names.

## Setup
Assumes performance code has been deployed under `stage-dal09-perf##client-##:/performance`

Cruiser_mon will automatically run against the carrier number that matches the perf-client. The tugboat service will run against ${carrier_number}00. The satellite service will run against satellite0. The hypershift support requires the kubeconfig for the management cluster to be available and the location specified in `startCruiserMonitoring.sh`

You can either use the following job -> https://alchemy-containers-jenkins.swg-devops.com/job/Armada-Performance/job/perfClient-setup/job/install-multiple-prereqs-on-perf-client/ and select `install-cruisermon.yaml` or install manually using the following instructions:

```
mkdir -p /performance/stats/cruiser_mon
sudo cp /performance/armada-perf/api/cruiser_mon/cruisermon*.service /lib/systemd/system
sudo cp /performance/armada-perf/api/cruiser_mon/cruisermon*.service /etc/systemd/system
sudo chmod 644 /etc/systemd/system/cruisermon*.service
```

To control cruisermon, you can either use this job -> https://alchemy-testing-jenkins.swg-devops.com/job/Armada-Performance/job/Utils/job/Control-CruiserMon-CruiserChurn/ or use the commands below directly on the perf-client.

```
sudo systemctl start cruisermon
sudo systemctl enable cruisermon
sudo systemctl start cruisermontugboat
sudo systemctl enable cruisermontugboat
sudo systemctl start cruisermonsatellite
sudo systemctl enable cruisermonsatellite
sudo systemctl start cruisermonhypershift
sudo systemctl enable cruisermonhypershift
```
Check cruiser_mon is running using:
```
sudo systemctl status cruisermon
sudo systemctl status cruisermontugboat
sudo systemctl status cruisermonsatellite
sudo systemctl status cruisermonhypershift
```
Once running and enabled it will automatically restart if the process unexpectedly terminates or the host reboots.

The logs will be written to `/performance/stats/cruiser_mon/cruiser_mon.log`

Results can be monitored at [Grafana](https://alchemy-prod.hursley.ibm.com/stage/performance/grafana/d/N6sEBkMWk/_cruiserchurn) (Select the Carrier you are interested in in the drop-down)

# Shutdown
Stop cruiser_mon using:
```
sudo systemctl stop cruisermon
sudo systemctl stop cruisermontugboat
sudo systemctl stop cruisermonsatellite
sudo systemctl stop cruisermonhypershift
```
(Note it will be started again on reboot unless you run):
```
sudo systemctl disable cruisermon
```
