#!/bin/bash -x
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Set up your KUBECONFIG before running this script
#export KUBECONFIG=<cruiser kube config file>

# Delete the range of pods from start to end, all inclusive
start=$1
end=$2

for ((i=$start;i<=$end;i++)); do
    echo deleting httpperf$i
    kubectl delete deployment -n httpperf$i httpperf
    kubectl delete services -n httpperf$i --all
done
