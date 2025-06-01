#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright Maximus Apen, 2025 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Set up your KUBECONFIG before running this script
#export KUBECONFIG=<cruiser kube config file>

# Run this script with command
#    nohup ./monitornode.sh &
# Check node status in node.log
# Remember to kill the monitornode.sh process after test

echo "Start test `date +%Y%m%d-%H%M%S`" > node.log
while [ 1 ]; do
    node=$(kubectl get nodes)
    echo "`date +%Y%m%d-%H%M%S`" >> node.log
    echo "$node" >> node.log
    #sleep 10
done
