#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to cleanup all resources for a partially deleted cruiser cluster
# Prereqs - Kubectl on carrier master

# This is last resort to remove all resources for a deleted cruiser cluster which are
#    - not listed in "ibmcloud ks clusters"
#    - "@Armada Xo - Stage cluster <cruiser cluster id>" in slack channel #armada-xo
#      shows ActualState as deleted
# but you see pods/resources for the cluster still running on carrier.
# Run this script on carrier to remove all resources related to the deleted cruiser.

if [ $# -lt 1 ]; then
    echo "Usage: cleanupDeletedCruiser.sh <cruiser cluster id>"
    exit 1;
fi

# Delete from all
# replicasets.extensions will be deleted indirectly so no need to include
for id in $(kubectl get all --all-namespaces | grep $1 | grep -v replicaset.apps | awk '{print $1":"$2}'); do
    # Need to use : in awk print instead of space, otherwise you get 2 separate id instead
    # Then replace : with space here
    id2=$(echo $id | sed "s/:/ /")
    kubectl delete -n $id2;
done

# Delete from configmap
for id in $(kubectl get cm --all-namespaces | grep $1 | awk '{print $1":"$2}'); do
    id2=$(echo $id | sed "s/:/ /")
    echo kubectl delete cm -n $id2
    kubectl delete cm -n $id2
done

# Delete from secret
for id in $(kubectl get secret --all-namespaces | grep $1 | awk '{print $1":"$2}'); do
    id2=$(echo $id | sed "s/:/ /")
    echo kubectl delete secret -n $id2
    kubectl delete secret -n $id2
done

# Delete from etcdcluster
for id in $(kubectl get etcdcluster --all-namespaces | grep $1 | awk '{print $1":"$2}'); do
    id2=$(echo $id | sed "s/:/ /")
    echo kubectl delete etcdcluster -n $id2
    kubectl delete etcdcluster -n $id2
done

# Delete from etcdbackup
for id in $(kubectl get etcdbackup --all-namespaces | grep $1 | awk '{print $1":"$2}'); do
    id2=$(echo $id | sed "s/:/ /")
    echo kubectl delete etcdbackup -n $id2
    kubectl delete etcdbackup -n $id2
done

# Delete from etcdrestore
for id in $(kubectl get etcdrestore --all-namespaces | grep $1 | awk '{print $1":"$2}'); do
    id2=$(echo $id | sed "s/:/ /")
    echo kubectl delete etcdrestore -n $id2
    kubectl delete etcdrestore -n $id2
done
