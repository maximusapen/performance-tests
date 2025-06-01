#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Disable NodeLocal DNS cache for a cluster

nodes=$(kubectl get nodes --no-headers | awk '{print $1}')
for node in ${nodes}; do
    kubectl label node $node --overwrite "ibm-cloud.kubernetes.io/node-local-dns-enabled=true"
done
kubectl get nodes -L "ibm-cloud.kubernetes.io/node-local-dns-enabled"
kubectl get pods -n kube-system -l k8s-app=node-local-dns -o wide
