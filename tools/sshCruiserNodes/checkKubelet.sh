#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright Maximus Apen, 2025 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Check to verify you can ssh to all nodes as root with your ssh private key
# Set up your KUBECONFIG before running this script
#export KUBECONFIG=<cruiser kube config file>

nodes=$(kubectl get nodes | grep -v NAME | awk '{print $1}')
echo $nodes

if [[ -z $nodes ]]; then
    # No workers found
    echo "No workers found"
    echo "Export KUBECONFIG before running this script."
    exit 1
fi

echo "`date +%Y%m%d-%H%M%S`"
for node in $nodes; do
    echo "`date +%Y%m%d-%H%M%S` Checking ssh to node $node as root. /root is returned"
    ssh -o StrictHostKeyChecking=no root@$node pwd
done
echo "End `date +%Y%m%d-%H%M%S`"
