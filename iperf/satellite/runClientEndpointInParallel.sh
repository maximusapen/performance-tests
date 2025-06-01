#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# Run specified number of iperfclient with specified satellite endpoints in parallel

if [[ $# -lt 4 ]]; then
    echo
    echo "Usage: ./runClientEndpointInParallel.sh <endpointPrefix> <cloud | location> <number_of_clients> <duration_in_seconds> "
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
    # Run in background so all clients starts at same time
    ./runClientEndpoint.sh ${endpointPrefix} ${endpointType} ${id} ${duration} &
done
