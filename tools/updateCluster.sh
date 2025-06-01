#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright Maximus Apen, 2025 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# Script to update the Kubernetes version for a cluster's master node
#
# Usage: updateCluster.sh <Cluster to update> [optional: Kube Version to update to]
# Update cluster master and workers Kubernetes version and monitor elapsed update time
#
if [[ "${BASH_VERSINFO:-0}" -lt 4 ]]; then
    echo "Script requires bash 4.0 or above"
    exit 1
fi

if [[ -z $KUBECONFIG ]]; then
    echo "Please ensure KUBECONFIG is set"
    exit 1
fi

perf_dir=/performance/armada-perf

if [[ $# -lt 1 ]]; then
    echo "Usage: $(basename $0) <cluster name> [kubernetes version]"
    exit 1
fi

clusterName=$1
k8sVersion=$2

${perf_dir}/tools/updateClusterMaster.sh ${clusterName} ${k8sVersion}
${perf_dir}/tools/updateClusterWorkers.sh ${clusterName}
