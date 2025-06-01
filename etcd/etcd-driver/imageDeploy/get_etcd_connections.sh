#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to extract etcd container connections

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
    echo "Usage: `basename $0` [<etcd_cluster> [<namespace>]]"
    echo "<etcd cluster> = The name of the etcd cluster."
    echo "<namespace> = The namespace to create etcd-drivers in"
    exit 1
fi

mkdir -p backup
dt=$(date +"%Y-%m-%d-%H-%M")

# Check the logs
echo "Etcd connections log label: ${dt}"
for c in `kubectl get pods -n ${namespace} --no-headers -l etcd_cluster=${etcd_cluster} | awk '{print $1}' `; do
    kubectl -n ${namespace} exec ${c} -c etcd -- netstat > backup/${c}.conns.${dt}.log
done

grep ESTABLISHED  backup/${etcd_cluster}*.conns.${dt}.log | awk '{print $5}' | sed -e "s/.*:172/172/g" | grep -v localhost | cut -d: -f1 | sort | uniq -c >  backup/${etcd_cluster}.conns.${dt}.log
echo "Etcd connections results: backup/${etcd_cluster}.conns.${dt}.log"


