#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# Extract logs from specified number of iperfclients into results directory
# If results directory already exists witth past data, those data will be deleted.
# If archiveDir is specified, data will be moved and archived in ${archiveDir}

if [[ $# -lt 1 ]]; then
    echo
    echo "Usage: ./getResult.sh <number_of_clients> [archive_directory_e.g._results_archive]"
    echo
    exit 1
fi

declare -i concurrency=$1
archiveDir=$2

if [[ -z ${CLIENT_KUBECONFIG} ]]; then
    echo "You need to export CLIENT_KUBECONFIG"
    exit 1
fi

echo Using CLIENT_KUBECONFIG ${CLIENT_KUBECONFIG}
export KUBECONFIG=${CLIENT_KUBECONFIG}

echo Checking IPerf3 pods
kubectl get pod -o wide | grep iperf

declare -i startpod=1
pods=$(kubectl get pods --no-headers | grep client | awk '{print $1}')
echo pods: ${pods}
set +e
mkdir -p results
rm results/* 2>/dev/null
set -e
declare -i nPod=0
for podName in ${pods}; do
    echo pod: ${podName}
    # Skip clients that may have been left from previous tests
    postfix=${podName#iperfclient-}
    if [[ ${postfix%%-*} -le $((concurrency + startpod - 1)) ]]; then
        #podName=$(echo $pods | cut -d$' ' -f1)
        echo "======== Detailed result from pod $podName are in results/${tstamp}_${podName} ========"
        kubectl logs ${podName} >results/${tstamp}_${podName}
    fi
    ((nPod++))
done

./parseresult.sh results

# Save data in specified archiveDir
if [[ ${archiveDir} != "" ]]; then
    echo Moving results from results directory to ${archiveDir}
    mkdir -p ${archiveDir}
    mv results/* ${archiveDir}
fi
