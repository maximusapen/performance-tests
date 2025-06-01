#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Set up your KUBECONFIG before running this script
#export KUBECONFIG=<cruiser kube config file>

# Run this script with command
#    nohup ./check_pleg.sh &
# Check for any pleg data in notready.log

echo "`date +%Y%m%d-%H%M%S`" > notready.log
while [ 1 ]; do
    notReadyNode=$(kubectl get nodes -o wide | grep NotReady)
    if [[ $notReadyNode != "" ]]; then
        echo "`date +%Y%m%d-%H%M%S`  $notReadyNode" >> notready.log
    fi
    notRunningPod=$(kubectl get pods --all-namespaces -o wide | grep httpperf | grep -v Running | grep -v NAME)
    if [[ $notRunningPod != "" ]]; then
        echo "`date +%Y%m%d-%H%M%S`  Pods not in Running state" >> notready.log
        echo "$notRunningPod" >> notready.log
    fi
    sleep 60
done
