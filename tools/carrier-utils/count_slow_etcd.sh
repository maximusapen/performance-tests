#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to count etcd warning messages about slow requests
# Prereqs - Kubectl with an appropriate KUBECONFIG set

if [[ $# -ne 1 && $# -ne 2 ]]; then
    echo "Usage: `basename $0` <node> <time_period> "
    echo "<node> = The node to count slow etcd messages for - or all to run against all nodes"
    echo "<time_period> = The time period to look in the logs for - e.g 1h = 1 hour"
    exit 1
fi

nod=$1
if [[ $nod == "all" ]]; then
    nodes=$(kubectl get nodes | grep " Ready " | awk '{print $1}')
else
    nodes=$nod
fi

if [[ $# == 2 ]]; then
    period=$2
else
    period=6h
fi
echo "Checking logs for last $period"

DATE=$(date -u +"%FT%H%M")
resultsFile=slowEtcdCounts_$DATE.csv


echo "Node,kubx-master pod count,kubx-etcd pod count,mean slow requests,mean slow wal sync" >$resultsFile

OIFS=$IFS
IFS=$'\n'

for node in $nodes; do

    nodeInfo=$(kubectl describe node $node)
    etcdPods=$(echo "$nodeInfo" | grep kubx-etcd)
    masterPods=$(echo "$nodeInfo" | grep kubx-masters)
    numEtcdPods=$(echo "$etcdPods" | wc -l)
    numMasterPods=$(echo "$masterPods" | wc -l)

    echo "Node $node has $numEtcdPods kubx-etcd pods and $numMasterPods kubx-master pods"

    totalSlow=0
    totalSlowWal=0
    for pod in $etcdPods; do
        namespace=$(echo ${pod}| awk '{print $1}')
        podName=$(echo ${pod}| awk '{print $2}')
        numSlow=$(kubectl logs $podName -n $namespace --since=$period | grep "took too long" | wc -l)
        numSlowWal=$(kubectl logs $podName -n $namespace --since=$period | grep "wal: sync duration of" | wc -l)
        echo "Found pod $podName in namespace $namespace, slow Requests: $numSlow, slow wal sync: $numSlowWal"
        totalSlow=$(($totalSlow + $numSlow))
        totalSlowWal=$((totalSlowWal + $numSlowWal))
    done
    meanSlow=$((totalSlow / $numEtcdPods))
    meanSlowWal=$(($totalSlowWal / $numEtcdPods))

    echo "Node $node has $numEtcdPods kubx-etcd pods and $numMasterPods kubx-master pods, meanSlowRequests: $meanSlow, meanSlowWal: $meanSlowWal"
    echo "$node,$numMasterPods,$numEtcdPods,$meanSlow,$meanSlowWal" >> $resultsFile
done
IFS=$OIFS
