#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# Run specified number of iperfclient with specified satellite endpoints in loops forever.
# Each client is restarted after running duration in seconds.

if [[ $# -lt 4 ]]; then
    echo
    echo "Usage: ./runClientEndpointLoop.sh <endpointPrefix> <cloud | location> <number_of_clients> <duration_in_seconds> "
    echo
    exit 1
fi

# Specified the endpointPrefix used in createtSatelliteEndpoint.sh
endpointPrefix=$1
# Endpoint Type can be location or cloud
endpointType=$2
# Number of iperfclient to run
declare -i nClient=$3
# Duration to run in seconds
declare -i duration=$4

for id in $(seq 1 ${nClient}); do
    echo Creating iperf3 client ${id}
    ./runClientEndpoint.sh ${endpointPrefix} ${endpointType} ${id} ${duration}
done

echo Using CLIENT_KUBECONFIG ${CLIENT_KUBECONFIG}
export KUBECONFIG=${CLIENT_KUBECONFIG}

getResultAndRestartClient() {
    id=$1
    echo "Restarting client $id"
    # Get completed result
    pods=$(kubectl get pods --no-headers | grep iperfclient-$id- | awk '{print $1}')
    for podName in ${pods}; do
        echo pod: ${podName}
        echo "======== Detailed result from pod $podName are in results/${tstamp}_${podName} ========"
        kubectl logs ${podName} >results/${tstamp}_${podName}
    done
    echo Moving results from results directory to results_loop
    mv results/* results_loop

    # Restart client id
    ./runClientEndpoint.sh ${endpointPrefix} ${endpointType} ${id} ${duration}
}

mkdir -p results
mkdir -p results_loop

while true; do
    echo "Sleep for a min"
    sleep 60
    # Check and restart
    for id in $(seq 1 ${nClient}); do
        echo Checking $id
        idPod=$(kubectl get pod | grep iperfclient-$id-)
        if [[ ${idPod} == "" ]]; then
            echo "Client is not running."
            getResultAndRestartClient $id
        fi
        if [[ ${idPod} == *"Error"* ]]; then
            echo "Client in Error state"
            if [[ ${idPod} != *"Running"* ]]; then
                echo "No Running client."
                getResultAndRestartClient $id
            fi
        fi
        if [[ ${idPod} == *"Completed"* ]]; then
            echo "Client completed."
            getResultAndRestartClient $id
        fi
    done
done
