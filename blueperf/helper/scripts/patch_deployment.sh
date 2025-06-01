#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

declare -A nodes
declare -i numNodes

allNodes=$(kubectl get node | grep -v NAME | awk '{print $1}')

numNodes=0
# Set up an array of nodes so we can round-robin the nodes
# when labeling nodes for services
for node in $allNodes; do
    echo node[$numNodes] = $node
    nodes[${numNodes}]=$node
    numNodes=numNodes+1
done
echo Number of nodes in cluster: ${numNodes}

# Ideally, we need 3 nodes for 3 db pods and extra pod(s) for services.
# Do not patch if cluster has less than 4 workers.

if [[ ${numNodes} < 4 ]]; then
   echo Cluster has only ${numNodes} workers.  Nodes will not be labeled and pods will not be patched.
   exit 0
fi

# See SECURE_USER_CALLS in https://github.com/blueperf/acmeair-mainservice-java/blob/master/Modes.md.
# Re-distribute customer-db flight-db pods will change signature and cause failure JWTVerifier failure.
# So not included in labels for moving pods
labels="customerservice flightservice authservice bookingservice"
bookingDBNode=""
customerDBNode=$(kubectl get pod -o wide | grep acmeair-customer-db | awk '{print $7}')
flightDBNode=$(kubectl get pod -o wide | grep acmeair-flight-db | awk '{print $7}')

declare -i i
declare -i count

# First, label node and pod for booking-db on node that does not have customer-db flight-db pods running
for node in $allNodes; do
    if [[ ${node} == $customerDBNode || ${node} == $flightDBNode ]]; then
        echo Not labeling this node ${node} for booking-db - one of customer-db and/or flight-db nodes
    else 
        # Node can be labeled.
        echo Label ${node} with booking-db
        kubectl patch node ${node} -p '{"metadata":{"labels":{"booking-db":"true"}}}'
        kubectl patch deployment acmeair-booking-db -p '{"spec":{"template":{"spec":{ "nodeSelector":{"booking-db":"true"}}}}}'
        bookingDBNode=${node}
        # Job done, now break
        break
    fi
done

if [[ ${bookingDBNode} == "" ]]; then
    echo Failed to label node for booking-db
    exit 1
fi

# Now label node for the services - avoiding booking-db nodes 
# but can share customer-db and flight-db nodes
i=0
for label in $labels; do
    if [[ ${nodes[$i]} == ${bookingDBNode} ]]; then
        echo Not labeling $label for this node ${nodes[$i]} which is reserved for booking-db
        # Advance to next node
        i=i+1
        if [[ $i == ${numNodes} ]]; then
            i=0
        fi
    fi
    
    # Node can be labeled.
    echo Label ${nodes[$i]} with ${label}
    kubectl patch node ${nodes[$i]} -p '{"metadata":{"labels":{"'${label}'":"true"}}}'

    i=i+1
    if [[ $i == ${numNodes} ]]; then
        i=0
    fi
done
kubectl get nodes --show-labels

# Now patch deployments to restart on labeled nodes
for label in $labels; do
    echo Deploy acmeair-${label} to node label ${label}
    kubectl patch deployment acmeair-${label} -p '{"spec":{"template":{"spec":{ "nodeSelector":{"'${label}'":"true"}}}}}'
done

echo
echo Sleeping for 240s for pods Running
sleep 240

kubectl get pod -o wide

# Check for not Running pods, most likely Pending but can be any other states
# Fail test if not all pods are Running. Pods from previous runs may still be Terminating - exclude.
badPods=$(kubectl get pods | grep -v Running | grep -v NAME | grep -v Terminating | awk '{print $1}')
if [[ "${badPods}" != "" ]]; then
    for badPod in ${badPods}; do
        echo
        echo badPod: $badPod
        kubectl describe pod ${badPod}
    done
    echo "Failing test as not all acmeair pods are Running.  See more details in describe pod(s) above."
    kubectl get pods -o wide
    exit 1
fi
