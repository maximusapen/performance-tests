#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to extract etcd member logs
# Prereqs - Kubectl with an appropriate KUBECONFIG set

if [[ $# -gt 0 ]]; then
    echo "Usage: `basename $0`"
    exit 1
fi

etcd_cluster=etcd-501-armada-stage5-south
namespace=armada
dt=$(date +"%Y-%m-%d-%H-%M")

# Check the logs
echo "${dt}"
for c in `kubectl get pods -n ${namespace} --no-headers -l etcd_cluster=${etcd_cluster} | awk '{print $1}' `; do
    kubectl -n ${namespace} logs ${c} -c etcd > backup/${c}.${dt}.log
done

