#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Set up your KUBECONFIG before running this script
#export KUBECONFIG=<cruiser kube config file>

nodes=$(kubectl get nodes | grep -v NAME | awk '{print $1}')
echo $nodes

echo "Start stopping kubelet on all nodes `date +%Y%m%d-%H%M%S`"
for node in $nodes; do
    echo "`date +%Y%m%d-%H%M%S` Stopping kubelet for node $node"
    ssh -o StrictHostKeyChecking=no root@$node sudo systemctl stop kubelet &
done
echo "End stop kubelet `date +%Y%m%d-%H%M%S`"
