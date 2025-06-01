#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Set up your KUBECONFIG before running this script
#export KUBECONFIG=<cruiser kube config file>

# Before running this script to copy the secret from httpperf1 namespace
# you need to create the httpperf1 with he performance-registry-token
# with initNamespace.sh

if [ $# -lt 2 ]; then
    echo "Usage: cloneNamespace <start> <end>"
    echo "  e.g. To create/clone namespaces httpperf2 to httpperf1000 from httpperf1 (created by initNamespace.sh)"
    echo "           cloneNamespace.sh 2 1000"

    exit 1
fi

# Parameters
start=$1 # Pod number to start, inclusive.
end=$2   # Pod number to end, inclusive

# Create all namespaces first.
for ((i = $start; i <= $end; i++)); do
    namespace=httpperf$i
    echo creating namespace $namespace
    kubectl create namespace $namespace
done

# Now copy the performance-registry-token from httpperf1
for ((i = $start; i <= $end; i++)); do
    namespace=httpperf$i
    echo "$(date +%Y%m%d-%H%M%S) copying registry token for $namespace"
    kubectl get secret performance-registry-token --namespace=httpperf1 --export -o yaml | kubectl apply --namespace=httpperf$1 -f -
    echo "$(date +%Y%m%d-%H%M%S) registry token created for $namespace"
done
