#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018, 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to delete etcd-operator clusters
# Prereqs - Kubectl with an appropriate KUBECONFIG set, and helm must be already
#  configured.

USAGE="Usage: delete_operator_clusters.sh <cruiser name prefix> <namespace>"

if [[ $# -ne 2 ]]; then
    echo $USAGE
    exit
fi

PREFIX=$1
NAMESPACE=$2

echo "Going to delete all etcd-operator clusters with prefix $PREFIX in namespace $NAMESPACE"

# First delete any etcd-driver deployments
for c in `helm list --namespace $NAMESPACE | grep $PREFIX | awk '{print $1}' `; do
    echo "Deleting etcd-driver $c in namespace $NAMESPACE"
    helm uninstall $c --namespace $NAMESPACE
done

for c in `kubectl get etcdclusters -n $NAMESPACE --no-headers | grep $PREFIX | awk '{print $1}' `; do
    echo "Deleting etcdcluster $c in namespace $NAMESPACE"
    kubectl delete etcdcluster $c -n $NAMESPACE
done

echo "Waiting for the etcdclusters to fully delete"
for (( i=0; i<20; i++ )); do
    kubectl -n $NAMESPACE get pods | grep $PREFIX
    if [[ $? -ne 0 ]]; then
        break
    fi
    sleep 5
done

# If the clusters didn't delete cleanly we need to delete the pods and Services
for c in `kubectl -n $NAMESPACE get pods | grep $PREFIX | awk '{print $1}' `; do
    echo "Deleting pod $c in namespace $NAMESPACE"
    kubectl delete pod $c -n $NAMESPACE
done

for c in `kubectl -n $NAMESPACE get services | grep $PREFIX | awk '{print $1}' `; do
    echo "Deleting service $c in namespace $NAMESPACE"
    kubectl delete service $c -n $NAMESPACE
done

# Delete the Secrets
for c in `kubectl -n $NAMESPACE get secrets | grep $PREFIX | awk '{print $1}' `; do
    echo "Deleting secret $c in namespace $NAMESPACE"
    kubectl delete secret $c -n $NAMESPACE
done
