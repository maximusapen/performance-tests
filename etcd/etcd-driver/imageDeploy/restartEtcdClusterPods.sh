#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Slowly restarts etcd pods in cluster. Primarily used to mimic nodes being rebooted.

. etcd-perftest-config

cluster_size=$(kubectl -n armada get pods -l etcd_cluster=${ETCDCLUSTER_NAME} --no-headers | grep Running | wc -l)
if [[ ${cluster_size} -lt 3 ]]; then
    echo "ERROR: Expected cluster of at least 3 nodes"
    exit 1
fi

echo "Start: $(date +"%Y-%m-%d-%H-%M")"

echo "Etcd-operator pods"
kubectl -n ${NAMESPACE} get pods -o wide | grep etcd-operator

echo "Etcd cluster pods"
kubectl -n ${NAMESPACE} get pods -l etcd_cluster=${ETCDCLUSTER_NAME} -o wide

for i in `kubectl -n ${NAMESPACE} get pods -l etcd_cluster=${ETCDCLUSTER_NAME} --no-headers | cut -d" " -f1`; do 
    node=$(kubectl -n ${NAMESPACE} get pods $i -o jsonpath='{.status.hostIP}')
    if [[ -n ${node} ]]; then
        echo "Draining ${node} to force restart of $i"
        kubectl drain ${node} --delete-emptydir-data --ignore-daemonsets
        echo "Sleeping for 10 minutes"
        # Relies on etcd-operator restarting pod which can take a while
        sleep 600
        kubectl uncordon ${node}
        sleep 60
        running=$(kubectl -n ${NAMESPACE} get pods -l etcd_cluster=${ETCDCLUSTER_NAME} --no-headers | grep Running | wc -l)
        if [[ ${running} -ne ${cluster_size} ]]; then
            echo "ERROR: Don't have ${cluster_size} running etcd pods"
            kubectl -n ${NAMESPACE} get pods -l etcd_cluster=${ETCDCLUSTER_NAME}
            exit 1
        fi
    else
        echo "Didn't find node to cordon for $i. Exiting script"
        exit 1
    fi
done

echo "Etcd cluster pods after restarts"
kubectl -n ${NAMESPACE} get pods -l etcd_cluster=${ETCDCLUSTER_NAME} -o wide

echo "Etcd-operator pods after restarts"
kubectl -n ${NAMESPACE} get pods -o wide | grep etcd-operator

echo "End: $(date +"%Y-%m-%d-%H-%M")"
