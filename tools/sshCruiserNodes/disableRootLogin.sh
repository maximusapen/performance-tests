#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Remove /root/.ssh from all workers 
# Set up your KUBECONFIG before running this script
#export KUBECONFIG=<cruiser kube config file>

nodes=$(kubectl get nodes --no-headers | awk '{print $1}')
echo $nodes

if [[ -z $nodes ]]; then
    # No workers found
    echo "No workers found"
    echo "Export KUBECONFIG before running this script."
    exit 1
fi

echo "`date +%Y%m%d-%H%M%S`"
for node in $nodes; do
    echo "`date +%Y%m%d-%H%M%S` ssh to node $node as root and disable root login."
    ssh -o StrictHostKeyChecking=no root@$node 'sed -i "s/PermitRootLogin.*/PermitRootLogin no/g" /etc/ssh/sshd_config; killall -1 sshd'
done
echo "End `date +%Y%m%d-%H%M%S`"
