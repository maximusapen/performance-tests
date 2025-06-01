#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# Run sequence of specified number of iperfclient.
# Each sequence run with specified satellite endpoints in parallel

# Modify the number of clients to suit your test
declare -a nClients=("1" "3" "6" "9" "10" "12" "15" "18")

if [[ $# -lt 3 ]]; then
    echo
    echo "Usage: ./runTest.sh <endpointPrefix> <cloud | location> <duration_in_seconds> "
    echo
    echo "Running these connections: ${nClients[@]}"
    echo "Modify nClients to change your test to suit"
    echo
    echo "You can check nohup.out for the results using nohup as follows:"
    echo "  nohup ./runTest.sh <endpointPrefix> <cloud | location> <duration_in_seconds> &"
    echo "Then search or grep for 'Total throughput' in nohup.out "
    exit 1
fi

echo "Running these connections: ${nClients[@]}"
echo "Modify nClients to change your test to suit."

echo Checking IPerf server pods
echo Using SERVER_KUBECONFIG ${SERVER_KUBECONFIG}
export KUBECONFIG=${SERVER_KUBECONFIG}
kubectl get pod -o wide | grep iperfserver

echo Using CLIENT_KUBECONFIG ${CLIENT_KUBECONFIG}
export KUBECONFIG=${CLIENT_KUBECONFIG}

set +e
echo "Removing old clients"
./rmIperf.sh client 10 2>/dev/null
set -e

# Specified the endpointPrefix used in createtSatelliteEndpoint.sh
endpointPrefix=$1
# Endpoint Type can be location or cloud
endpointType=$2
# Duration to run in seconds
declare -i duration=$3

retries=5
for id in ${nClients[@]}; do
    for i in $(seq 1 ${retries}); do
        echo "Retry $i: Calling runClient $id"
        ./runClientEndpointInParallel.sh ${endpointPrefix} ${endpointType} ${id} ${duration}
        sleepTime=$((${duration} + 300))
        echo "Sleeping for ${sleepTime}"
        sleep ${sleepTime}
        clientPods=$(kubectl get pod -o wide | grep iperfclient)
        if [[ ${clientPods} == *"Error"* ]]; then
            echo "Error found.  Checking Error"
            ./getResult.sh ${id}
            echo "Remove client"
            ./rmIperf.sh client $id
            if [[ $i -ne ${retries} ]]; then
                echo "Retry run again"
            fi
            continue
        else
            # No Errors
            ./getResult.sh ${id} results_${endpointType}_${id}
            echo "Remove client"
            ./rmIperf.sh client $id
            break
        fi
    done
done
