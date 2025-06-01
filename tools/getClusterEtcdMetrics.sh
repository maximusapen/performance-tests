#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# Script to gather etcd metrics from a set of cluster masters on a carrier
#
# Usage: getClusterEtcdMetrics.sh [<filter>]
# <filter> is optional - if omitted stats will be gathered from all masters on the carrier
# This needs to be run on the master for the carrier (or at least somewhere that has access
# to kubectl and the certificates for the masters).

filter=$1
if [ -z "$filter" ]
then
  filter="."
fi
startTime=$(date)
echo "$startTime : Gathering etcd metric for all masters that match a grep with $filter"

IFS=$'\n'
host=$(hostname)
statsFile=etcdStats.txt
resultsFile=clusterEtcdStats_$host.csv
echo "Carrier master, Cluster Name, Node, Etcd Start time, Etcd Running duration (s), Key Count, DB Size, Deletes, Deletes per Sec, Puts, Puts per Sec, Ranges, Ranges per Sec, Txns, Txns per Sec, Events, Events per Sec, Slow watchers, Watch Streams, Watchers, Compact DB Time Total (s), Compact DB Count, Compact DB Time Mean (s), Compact Index Time Total (s), Compact Index Count, Compact Index Time Mean (s), Fsync Time Total (s), Fsync Count, Fsync Time Mean (s)" > $resultsFile
for pods in $(kubectl get pods -n kubx-masters --no-headers | grep Running | grep $filter)
do
    podName=$(echo $pods |cut -d$' ' -f1)
    # Need to trim off master and last 2 parts of pod name - this is a bit more complex than it
    # really needs to due to the fact that our fake masters use multiple '-' chars in name
    clusterName=$(echo $podName| cut -d$'-' -f2- |rev | cut -d$'-' -f3- | rev)
    podInfo=$(kubectl get pod $podName -n kubx-masters -o=json)
    #endpoint=$(echo $podInfo | grep 'advertise-client-urls' | cut -d$'=' -f2)
    etcdStartTime=$(echo $podInfo | jq -r '.status.containerStatuses[] | select(.name == "etcd") | .state.running.startedAt')
    node=$(echo $podInfo | jq -r .spec.nodeName)
    endpoint=$(kubectl describe pod $podName -n kubx-masters | grep 'advertise-client-urls' | cut -d$'=' -f2)

    #etcdStartTime=$(kubectl get pod $podName -n kubx-masters -o=json | jq -r '.status.containerStatuses[] | select(.name == "etcd") | .state.running.startedAt')
    if [ -n "$etcdStartTime" ]
    then
        etcdRunningTime=$(($(date +%s) - $(date -d $etcdStartTime +%s)))
    fi
    if [[ $endpoint ]]
    then
         curl -L --cacert /mnt/nfs/$clusterName/etc/kubernetes/cert/ca.pem --cert /mnt/nfs/$clusterName/etc/kubernetes/cert/etcd.pem --key /mnt/nfs/$clusterName/etc/kubernetes/cert/etcd-key.pem $endpoint/metrics > $statsFile
         keyCount=$(cat $statsFile |grep ^etcd_debugging_mvcc_keys_total | cut -d$' ' -f2 | xargs printf '%.0f')
         dbSize=$(cat $statsFile | grep ^etcd_debugging_mvcc_db_total_size_in_bytes | cut -d$' ' -f2 | xargs printf '%.0f')
         deletes=$(cat $statsFile |grep ^etcd_debugging_mvcc_delete_total | cut -d$' ' -f2 | xargs printf '%.0f')
         deletesRate=$(awk -v a=$deletes -v b=$etcdRunningTime 'BEGIN { print (a/b) }' | xargs printf '%.2f')
         events=$(cat $statsFile |grep ^etcd_debugging_mvcc_events_total | cut -d$' ' -f2 | xargs printf '%.0f')
         eventsRate=$(awk -v a=$events -v b=$etcdRunningTime 'BEGIN { print (a/b) }' | xargs printf '%.2f')
         puts=$(cat $statsFile |grep ^etcd_debugging_mvcc_put_total | cut -d$' ' -f2 | xargs printf '%.0f')
         putsRate=$(awk -v a=$puts -v b=$etcdRunningTime 'BEGIN { print (a/b) }' | xargs printf '%.2f')
         range=$(cat $statsFile |grep ^etcd_debugging_mvcc_range_total | cut -d$' ' -f2 | xargs printf '%.0f')
         rangeRate=$(awk -v a=$range -v b=$etcdRunningTime 'BEGIN { print (a/b) }' | xargs printf '%.2f')
         slow_watchers=$(cat $statsFile |grep ^etcd_debugging_mvcc_slow_watcher_total | cut -d$' ' -f2 | xargs printf '%.0f')
         txn=$(cat $statsFile |grep ^etcd_debugging_mvcc_txn_total | cut -d$' ' -f2 | xargs printf '%.0f')
         txnRate=$(awk -v a=$txn -v b=$etcdRunningTime 'BEGIN { print (a/b) }' | xargs printf '%.2f')
         watch_stream=$(cat $statsFile |grep ^etcd_debugging_mvcc_watch_stream_total | cut -d$' ' -f2 | xargs printf '%.0f')
         watcher=$(cat $statsFile |grep ^etcd_debugging_mvcc_watcher_total | cut -d$' ' -f2 | xargs printf '%.0f')

         compactTime=$(cat $statsFile | grep ^etcd_debugging_mvcc_db_compaction_pause_duration_milliseconds_sum | cut -d$' ' -f2 | xargs printf '%.0f')
         compactCount=$(cat $statsFile | grep ^etcd_debugging_mvcc_db_compaction_pause_duration_milliseconds_count | cut -d$' ' -f2 | xargs printf '%.0f')
         if [[ $compactCount != '0' ]]; then
             compactMean=$(awk -v a=$compactTime -v b=$compactCount 'BEGIN { print (a/b) }' | xargs printf '%.4f')
         else
             compactMean=""
         fi

         compactIndexTime=$(cat $statsFile | grep ^etcd_debugging_mvcc_index_compaction_pause_duration_milliseconds_sum | cut -d$' ' -f2 | xargs printf '%.0f')
         compactIndexCount=$(cat $statsFile | grep ^etcd_debugging_mvcc_index_compaction_pause_duration_milliseconds_count | cut -d$' ' -f2 | xargs printf '%.0f')
         if [[ $compactIndexCount != '0' ]]; then
             compactIndexMean=$(awk -v a=$compactIndexTime -v b=$compactIndexCount 'BEGIN { print (a/b) }' | xargs printf '%.4f')
         else
            compactIndexMean=""
         fi

         fsyncTime=$(cat $statsFile | grep ^etcd_disk_wal_fsync_duration_seconds_sum | cut -d$' ' -f2 | xargs printf '%.0f')
         fsyncCount=$(cat $statsFile | grep ^etcd_disk_wal_fsync_duration_seconds_count | cut -d$' ' -f2 | xargs printf '%.0f')
         if [[ $fsyncCount != '0' ]]; then
             fsyncMean=$(awk -v a=$fsyncTime -v b=$fsyncCount 'BEGIN { print (a/b) }' | xargs printf '%.4f')
         else
             fsyncMean=""
         fi

         echo "$host,$clusterName,$node,$etcdStartTime,$etcdRunningTime,$keyCount,$dbSize,$deletes,$deletesRate,$puts,$putsRate,$range,$rangeRate,$txn,$txnRate,$events,$eventsRate,$slow_watchers,$watch_stream,$watcher,$compactTime,$compactCount,$compactMean,$compactIndexTime,$compactIndexCount,$compactIndexMean,$fsyncTime,$fsyncCount,$fsyncMean" >> $resultsFile
         rm $statsFile
    fi
done
endTime=$(date)
echo "$endTime : Completed"
