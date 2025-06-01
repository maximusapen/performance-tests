#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# Script to update the Kubernetes version for a cluster's master node
#
# Usage: updateClusterMaster.sh <Cluster to update> [optional: Kube Version to update to]
# Update cluster master Kubernetes version and monitor elapsed update time
#

perf_dir=/performance

if [[ $# -lt 1 ]]; then
    echo "Usage: $(basename $0) <cluster name> [kubernetes version]"
    exit 1
fi
clusterName=$1
k8sVersion=$2

if [ -n ${k8sVersion} ]; then
    k8sVersionStr="-kubeVersion=${k8sVersion}"
else
    k8sVersion="default"
fi

# Ensure we're currently in Ready state
while
    clusterInfo=$(${perf_dir}/bin/armada-perf-client -action=GetCluster -clusterName=${clusterName} | awk 'NR>2{print;}' | sed \$d)
    masterStatus=$(echo ${clusterInfo} | jq -r '.masterStatus')
    origMasterStatusModifiedDate=$(echo ${clusterInfo} | jq -r '.masterStatusModifiedDate')
    ((${masterStatus} != "Ready"))
do
    printf "%s - Waiting for master to be 'Ready'.\n" "$(date +%T)"
    sleep 120
done

printf "%s - Updating cluster master to %s\n" "$(date +%T)" "${k8sVersion}"

# Start time in seconds since epoch
startTime=$(date +%s)

# Request master update to version specified, or default target version if none specified
${perf_dir}/bin/armada-perf-client -action=UpdateCluster -clusterName=${clusterName} ${k8sVersionStr}

# Wait for update to complete
while
    clusterInfo=$(${perf_dir}/bin/armada-perf-client -action=GetCluster -clusterName=${clusterName} | awk 'NR>2{print;}' | sed \$d)
    masterStatus=$(echo ${clusterInfo} | jq -r '.masterStatus')
    masterStatusModifiedDate=$(echo ${clusterInfo} | jq -r '.masterStatusModifiedDate')
    [[ ${masterStatus} != "Ready" || ${masterStatusModifiedDate} == ${origMasterStatusModifiedDate} ]]
do
    printf "%s - Waiting for cluster update to complete'.\n" "$(date +%T)"
    sleep 120
done

# Update complete
printf "%s - Master status now '%s', modified at %s\n" "$(date +%T)" "${masterStatus}" "${masterStatusModifiedDate}"

# Use the master status modifed date to determine when the update completed
# We'll convert into seconds since the epoch to allow us to calculate the total duration
endTime=$(date -jf '%Y-%m-%dT%H:%M:%S' '+%s' ${masterStatusModifiedDate%+*})
duration=$((endTime - startTime))

printf "\n%s - '%s' master update complete. Total Duration: %ss.\n" "$(date +%T)" "${clusterName}" "${duration}"
