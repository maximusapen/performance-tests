#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018, 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Set up your KUBECONFIG before running this script
#export KUBECONFIG=<cruiser kube config file>

# Create the range of pods from start to end, all inclusive

if [ $# -lt 3 ]; then
    echo "Usage: createtestpods.sh <start> <end> <start port number between 30000 and 32767>"
    echo "  e.g. To create a test pod in namespaces from httpperf1 to httpperf1000 with port numnber starting 30000:"
    echo "           createtestpods.sh 1 1000 30000"

    exit 1
fi

start=$1
end=$2
port=$3

for ((i = $start; i <= $end; i++)); do
    echo creating $i

    HTTPPERF_NP_HTTP=$port
    port=$((port + 1))
    HTTPPERF_NP_HTTPS=$port
    port=$((port + 1))
    HTTPPERF_LB_HTTP=$port
    port=$((port + 1))
    HTTPPERF_LB_HTTPS=$port
    port=$((port + 1))
    echo next port $port

    sed "s/HTTPPERF_NP_HTTP/${HTTPPERF_NP_HTTP}/" deployTemplate.yaml | sed "s/HTTPPERF_NP_HTTPS/${HTTPPERF_NP_HTTPS}/" | sed "s/HTTPPERF_LB_HTTP/${HTTPPERF_LB_HTTP}/" | sed "s/HTTPPERF_LB_HTTPS/${HTTPPERF_LB_HTTPS}/" >deployPod.yaml

    kubectl create -f deployPod.yaml -n httpperf$i

done
