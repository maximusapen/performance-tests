#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright Maximus Apen, 2025 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Run on carrier master to trigger and monitor master restartof one cluster

# Set cluster id of cluster below:

clusterId=$1

# Following KUBECONFIG on performance client.  Modify as appropriate.
carrier=carrier4
export KUBECONFIG=/performance/config/${carrier}_stage/admin-kubeconfig

# Scale master to one master pod
kubectl scale deployment -n kubx-masters master-${clusterId} --replicas=1

timestamp=$(date +%Y-%m-%d_%T)
echo "Restart ${clusterId}"
echo "Start Time ${timestamp}"

# Change the value for PERF_TESTING for repeat runs to clusters to ensure master is restarted
kubectl patch deploy -n kubx-masters master-${clusterId} -p '{"spec":{"template":{"spec":{"containers":[{"name":"kms","env":[{"name":"PERF_TESTING","value":"'${timestamp}'"}]}]}}}}'
echo "Time patched $(date +%Y-%m-%d_%T)"
echo "Wait for all 5 containers in Running state ...."
SECONDS=0
while true; do
    mpod=$(kubectl get pod -n kubx-masters -o wide | grep master-${clusterId})
    if [[ ${mpod} != *"Terminating"* && ${mpod} == *"Running"* && ${mpod} == *"5/5"* ]]; then
        running=true
        break
    fi
done
echo "Time master restarted $(date +%Y-%m-%d_%T)"
echo "Time to restart master pod: ${SECONDS} sec"
kubectl get pod -n kubx-masters -o wide | grep master-${clusterId}
