#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018, 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Requires ENV variable ${STAGE_GLOBAL_ARMPERF_IBMCLOUD_APIKEY} has been exported prior to running the script.

USAGE="Usage: etcd_mon.sh <cluster name prefix> <stats_interval> <post_metrics (true or false)>"

if [[ $# -ne 3 ]]; then
    echo $USAGE
    exit
fi

PREFIX=$1
INTERVAL=$2
PUBLISH=$3
METRICS_TOML=/performance/armada-perf/metrics/bluemix/metrics.toml
PERF_TOML=/performance/armada-perf/api/config/perf.toml

apiKey=${STAGE_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}
scheme=$(grep -w "scheme" $METRICS_TOML | cut -d$'=' -f2 | tr -d '[:space:]' |  tr -d '"')
root=$(grep -w "root" $METRICS_TOML | cut -d$'=' -f2 | tr -d '[:space:]' |  tr -d '"')
host=$(grep -w "host" $METRICS_TOML | cut -d$'=' -f2 | tr -d '[:space:]' |  tr -d '"')
path=$(grep -w "path" $METRICS_TOML | cut -d$'=' -f2 | tr -d '[:space:]' |  tr -d '"')

while true; do
    clusterCount=$(kubectl get etcdclusters --all-namespaces --no-headers | grep $PREFIX | wc -l)
    pods=$(kubectl get pods --all-namespaces -l app=etcd --no-headers | grep $PREFIX)
    runningPods=$(echo "$pods" | grep " Running " | wc -l)
    creatingPods=$(echo "$pods" | grep -E "Init:|PodInitializing|Pending|ContainerCreating" | wc -l)
    failedPods=$(echo "$pods" | grep -E "Error|CrashLoop|Terminating" | wc -l)
    podsMissing=$((($clusterCount * 3) - ($runningPods + $creatingPods + $failedPods)))
    echo "$DATE : Count: $clusterCount, Pods Running: $runningPods, Pods creating: $creatingPods ,Pods failed: $failedPods, Pods missing: $podsMissing"
    if [[ $PUBLISH == "true" ]]; then
        curl -XPOST --header "X-Auth-User-Token: apikey $apiKey" -d "[{\"name\" : \"dal09.${root}.EtcdOperatorClusterMon.Num_Clusters.count\",\"value\" : ${clusterCount}}]" ${scheme}://${host}${path}
        curl -XPOST --header "X-Auth-User-Token: apikey $apiKey" -d "[{\"name\" : \"dal09.${root}.EtcdOperatorClusterMon.PodsRunning.count\",\"value\" : ${runningPods}}]" ${scheme}://${host}${path}
        curl -XPOST --header "X-Auth-User-Token: apikey $apiKey" -d "[{\"name\" : \"dal09.${root}.EtcdOperatorClusterMon.PodsCreating.count\",\"value\" : ${creatingPods}}]" ${scheme}://${host}${path}
        curl -XPOST --header "X-Auth-User-Token: apikey $apiKey" -d "[{\"name\" : \"dal09.${root}.EtcdOperatorClusterMon.PodsFailed.count\",\"value\" : ${failedPods}}]" ${scheme}://${host}${path}
        curl -XPOST --header "X-Auth-User-Token: apikey $apiKey" -d "[{\"name\" : \"dal09.${root}.EtcdOperatorClusterMon.PodsMissing.count\",\"value\" : ${podsMissing}}]" ${scheme}://${host}${path}
    fi
    sleep $INTERVAL
done
