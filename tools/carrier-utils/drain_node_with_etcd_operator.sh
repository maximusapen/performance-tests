#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to cordon/drain a node which contains etcd-operator controlled etcd pods (These need to be manually deleted)
# Prereqs - Kubectl with an appropriate KUBECONFIG set

if [[ $# -ne 2 ]]; then
    echo "Usage: `basename $0` <node> <reason>"
    echo "<node> = The node to drain"
    echo "<reason> = The reason for draining"
    exit 1
fi

NODES=$1
REASON=$2
FAILURES=0

function drain_node() {
    NODE=$1
    echo "Going to cordon and drain node $NODE"

    kubectl cordon $NODE

    kubectl annotate node $NODE --overwrite node.cordon.reason="$REASON"

    IFS=$'\n'

    # If the clusters didn't delete cleanly we need to delete the pods and Services
    local pod
    for pod in `kubectl get pods --all-namespaces -o=wide --no-headers  | grep "kubx-etcd"| grep -w $NODE`; do
        local namespace=$(echo ${pod}| awk '{print $1}')
        local podName=$(echo ${pod}| awk '{print $2}')
        echo "Deleting pod $podName in namespace $namespace"
        kubectl delete pod $podName -n $namespace
    done

    echo "Waiting for the etcds to fully delete"
    local allDeleted=0
    local i=0
    for (( ; i<10; i++ )); do
        kubectl get pods --all-namespaces -o=wide --no-headers  | grep "kubx-etcd"| grep -w $NODE
        if [[ $? -ne 0 ]]; then
            allDeleted=1
            break
        fi
        sleep 30
    done

    if [[ $allDeleted -eq 1 ]]; then
        kubectl drain $NODE --force --timeout 420s --ignore-daemonsets --delete-emptydir-data
    else
        echo "Failed to delete all etcd pods withing the time limit - ensure all etcd pods are deleted before retrying the drain"
        FAILURES=$((FAILURES+1))
    fi
}

OIFS="$IFS"
IFS=","
for n in ${NODES}; do
    # Switching back handles problems with EXTRA_DEPLOY_PARAMS
    drain_node $n
    IFS=","
done
IFS="$OIFS"

exit $FAILURES
