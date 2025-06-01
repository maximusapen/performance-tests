#!/bin/bash

# Set up your KUBECONFIG before running this script
#export KUBECONFIG=<cruiser kube config file>

# Create the range of pods from start to end, all inclusive

if [ $# -lt 2 ]; then
    echo "Usage: createsimple.sh <start> <end>"
    echo "  e.g. To create a test pod in namespaces from httpperf1 to httpperf1000:"
    echo "           createtsimple.sh 1 1000"

    exit 1
fi

start=$1
end=$2

for ((i = $start; i <= $end; i++)); do
    echo creating $i

    kubectl create -f deploySimple.yaml -n httpperf$i

done
