#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to extract etcd member logs

. etcd-perftest-config

etcd_cluster=${ETCDCLUSTER_NAME}
namespace=${NAMESPACE}

if [[ $# -ge 1 ]]; then
    etcd_cluster=$1
fi

if [[ $# -eq 2 ]]; then
    namespace=$2
fi

if [[ $# -gt 2 ]]; then
    echo "Usage: `basename $0` [<etcd cluster> [<namespace>]]"
    echo "<etcd cluster> = The name of the etcd-operator instances."
    echo "<namespace> = The namespace to create etcd-drivers in"
    exit 1
fi

dt=$(date +"%Y-%m-%d-%H-%M")

# Check the logs
echo "Etcd log label: ${dt}"
for c in `kubectl get pods -n ${namespace} --no-headers -l etcd_cluster=${etcd_cluster} | awk '{print $1}' `; do
    kubectl -n ${namespace} logs ${c} -c etcd > backup/${c}.${dt}.log
done

echo "${dt}" > etcd_logs_label.txt
