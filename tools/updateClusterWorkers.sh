#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# Script to update the Kubernetes version for a cluster's worker nodes
#
# Usage: updateClusterWorkers.sh <cluster to update>
# Update cluster workers Kubernetes version and monitor elapsed update time
# Will use the default of no nore than 20% of workers to be updated in parallel
#

if [[ "${BASH_VERSINFO:-0}" -lt 4 ]]; then
    echo "Script requires bash 4.0 or above"
    exit 1
fi

if [[ -z $KUBECONFIG ]]; then
    echo "Please ensure KUBECONFIG is set"
    exit 1
fi

perf_dir=/performance

if [[ $# -lt 1 ]]; then
    echo "Usage: $(basename $0) <cluster name>"
    exit 1
fi
clusterName=$1

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

# Display current list of workers
ibmcloud ks workers --cluster "${clusterName}"

printf "\n%s - Updating cluster workers\n" "$(date +%T)"

# Generate comma separated list of workers in the cluster
workerList=$(${perf_dir}/bin/armada-perf-client -action=GetClusterWorkers -clusterName="${clusterName}" | awk 'NR>2{print;}' | sed \$d | jq -r '.[] | .id')
workers=$(echo -n ${workerList} | sed 's/ /,/g')

# Start time in seconds since epoch
startTime=$(date +%s)

# Request update of all cluster workers (no more than 20% at a time - the default)
ibmcloud ks worker update --cluster "${clusterName}" -w "${workers}" -f

# Monitor update process

while
    targetArr=($(${perf_dir}/bin/armada-perf-client -action=GetClusterWorkers -clusterName="${clusterName}" | awk 'NR>2{print;}' | sed \$d | jq -r '.[] | .targetVersion'))
    currentArr=($(${perf_dir}/bin/armada-perf-client -action=GetClusterWorkers -clusterName="${clusterName}" | awk 'NR>2{print;}' | sed \$d | jq -r '.[] | .kubeVersion'))
do
    declare -A nodeStatusMap=()
    nodeStatusArr=($(kubectl get nodes --no-headers | awk '{print $2":"$5}'))
    for ns in "${nodeStatusArr[@]}"; do
        ((nodeStatusMap[${ns}]++))
    done

    printf "%s - " "$(date +%T)"
    nodeStatusLen=${#nodeStatusArr[@]}
    nodeStatusMapLen=${#nodeStatusMap[@]}

    i=0
    for n in "${!nodeStatusMap[@]}"; do
        ((i++))
        printf "%s - %s/%s" "${n}" "${nodeStatusMap[$n]}" "${nodeStatusLen}"
        if ((i != nodeStatusMapLen)); then
            printf ", "
        fi
    done
    printf "\n"

    workersComplete=0
    for ((i = 0; i < ${#currentArr[@]}; i++)); do
        if [[ "${targetArr[i]}" == "${currentArr[i]}" ]]; then
            ((workersComplete++))
        fi
    done

    if ((workersComplete == nodeStatusLen)); then
        break
    fi

    sleep 60
done

# Update complete
endTime=$(date +%s)

duration=$((endTime - startTime))

printf "\n%s - '%s' worker updates complete. Total Duration: %ss.\n" "$(date +%T)" "${clusterName}" "${duration}"
