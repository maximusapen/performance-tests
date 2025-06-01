#!/bin/bash

# Delete all etcd-operator clusters in a carrier that don't have any pods.
# Prereqs - Kubectl with an appropriate KUBECONFIG set, and helm must be already
#  configured.

OIFS=$IFS
IFS=$'\n'

for cluster_details in `kubectl get etcdclusters --all-namespaces -o wide --no-headers`; do
    namespace=`echo $cluster_details | awk '{print $1}'`
    cluster=`echo $cluster_details | awk '{print $2}'`
    cnt=$(kubectl -n $namespace get pods -l app=etcd,etcd_cluster=$cluster --no-headers | wc -l | awk '{print $1}')
    if [[ $cnt -eq 0 ]]; then
        echo "No pods in $cluster"
        # Attempt to delete any etcd-driver instances for the etcd-operator
        helm uninstall $cluster-etcd-driver --namespace $namespace

        # Delete the etcdcluster
        kubectl -n $namespace delete etcdclusters $cluster
    fi
done

IFS=$OIFS
