#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# Script to Find and restore etcd clusters that have lost quorum
#
# Usage: restoreEtcds.sh [<filter>]
# <filter> is optional - if omitted all masters on the carrier will be checked
# This needs to be run on the master for the carrier.
#
# It will restore all etcds that have a master in CrashLoopBackOff & less than 2 etcd pods in Running state.
filter=$1
if [ -z "$filter" ]
then
  filter="."
fi
startTime=$(date)
echo "$startTime : Checking and restoring all etcds that match a grep with $filter"

OIFS=$IFS
IFS=$'\n'

brokenMasters=()
for pods in $(kubectl get pods -n kubx-masters --no-headers | grep CrashLoopBackOff | grep $filter)
do
    podName=$(echo $pods |cut -d$' ' -f1)
    # Need to trim off master and last 2 parts of pod name - this is a bit more complex than it
    # really needs to due to the fact that our fake masters use multiple '-' chars in name
    clusterName=$(echo $podName| cut -d$'-' -f2- |rev | cut -d$'-' -f3- | rev)

    if [[ ! " ${brokenMasters[@]} " =~ " ${clusterName} " ]]; then
      brokenMasters+=(${clusterName})
    fi
done

allpods=$(kubectl get pods --all-namespaces --show-all --no-headers)

count=0
for i in "${brokenMasters[@]}"
do
   echo "$i"
   running_etcd_pods=$(echo "$allpods" | grep "kubx-etcd-" | grep "$i-" | grep Running | wc -l)
   if [[ running_etcd_pods -lt 2 ]]; then
    echo "etcd for ${i} has ${running_etcd_pods} running etcd pods so will restore it"
    etcd_namespace=$(echo "$allpods" | grep "kubx-etcd-" | grep -m 1 "$i-" | awk '{print $1}')
    echo "Restoring etcd for ${i} in namespace ${etcd_namespace}"
    SECONDS=0
    armada-restore-cluster-etcd ${i} ${etcd_namespace}
    restoreTime=$SECONDS
    ((count+=1))
    echo "Restored etcd cluster for ${i} in ${restoreTime} seconds"
   else
    echo "etcd for ${i} has ${running_etcd_pods} running etcd pods so won't restore it"
   fi
done

endTime=$(date)
echo "$endTime : Finished Restoring ${count} etcds that matched a grep with $filter"
