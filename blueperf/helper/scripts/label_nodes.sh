#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

declare -a nodes=($(kubectl get node | grep -v NAME | awk '{print $1}'))
numNodes=${#nodes[@]}

echo "${nodes[@]}"
echo Number of nodes in cluster: ${numNodes}

# Ideally, we need 3 nodes for 3 db pods and extra pod(s) for services.
# Do not patch if cluster has less than 4 workers.

if [[ ${numNodes} < 4 ]]; then
    echo Cluster has only ${numNodes} workers. Nodes will not be labeled and pods will not be patched.
    exit 0
fi

labels="authservice customerservice flightservice bookingservice"

# Assign node[0] to booking-db
echo Label ${nodes[0]} with booking-db
kubectl patch node ${nodes[0]} -p '{"metadata":{"labels":{"booking-db":"true"}}}'

# Assign node[1] to customer-db
echo Label ${nodes[1]} with customer-db
kubectl patch node ${nodes[1]} -p '{"metadata":{"labels":{"customer-db":"true"}}}'

# Assign node[2] to flight-db
echo Label ${nodes[2]} with flight-db
kubectl patch node ${nodes[2]} -p '{"metadata":{"labels":{"flight-db":"true"}}}'

declare -i i
declare -i count

# Now assign services from node[3] onwards, round robin back to node[3]
# if services need to share nodes
i=3
for label in $labels; do
    echo Label ${nodes[i]} with ${label}
    kubectl patch node ${nodes[i]} -p '{"metadata":{"labels":{"'${label}'":"true"}}}'

    i=i+1
    if [[ $i == ${numNodes} ]]; then
        i=3
    fi
done
kubectl get nodes --show-labels
