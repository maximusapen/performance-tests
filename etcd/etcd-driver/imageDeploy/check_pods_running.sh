#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to stop etcd-driver test and collect results
# Prereqs - Kubectl with an appropriate KUBECONFIG set, and helm must be already configured.

if [[ $# -lt 2 ]]; then
    echo "Usage: `basename $0` <prefix> <namespace> [helm charg]"
    echo "<prefix> = The prefix to match etcd-operator instances."
    echo "<namespace> = The namespace to create etcd-drivers in"
    echo "[<helm chart>] = The helm chart (defaults to etcd-driver)"
    exit 1
fi

prefix=$1
namespace=$2
helm_chart=etcd-driver
if [[ $# -ge 3 ]]; then
    helm_chart=$3
fi  

# Check the logs
for c in `kubectl get pods -n ${namespace} --no-headers -l app=${helm_chart} | awk '{print $1}' `; do
    echo "Checking ${c} ==========================================="
    kubectl -n ${namespace} logs ${c} | egrep "Created test end watch for|Test End detected"
done

