#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019, 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Run on carrier master to make config changes to all masters for master restart
#
# If only KP clusters exists on carrier, you can use openvpnserver pods to get the KP cluster list
#clusterIds=$(kubectl get pod -n kubx-masters | grep openvpnserver | sed "s/-/ /g" | awk '{print $2}')

# Search for 5/5 master pods if non-KP clusters also exists on carrier

source ../test_conf.sh

# Optional id for logging purpose
secretID=$1

echo Expected number of KP master pods for ${numCluster} clusters: ${numMpods}

clusterIds=$(ibmcloud ks clusters | grep kpcluster | awk '{print $2}')

export KUBECONFIG=/performance/config/${carrier}_stage/admin-kubeconfig

date
echo "Scaling down all clusters"
#theValue=$(date +%Y-%m-%d_%T)
for clusterId in ${clusterIds}; do
    echo Scaling down ${clusterId}
    # Change the value for PERF_TESTING for repeat runs to clusters to ensure master is restarted
    #kubectl patch deploy -n kubx-masters master-${clusterId} -p '{"spec":{"template":{"spec":{"containers":[{"name":"kms","env":[{"name":"PERF_TESTING","value":"'${theValue}'"}]}]}}}}'
    kubectl scale deployment -n kubx-masters master-${clusterId} --replicas=0
    #sleep 1
done

date
echo "Check all clusters terminated"
# Wait for all clusters to scale down to 0
# If one or more master pods are stuck in Terminating state with error, then force the Terminating pods with:
#        kubectl -n kubx-masters delete pod <master pod> --force=true --grace-period=0
while true; do
    nPods=$(kubectl get pod -n kubx-masters | grep "/5" | wc -l)
    echo "Number of master pods: ${nPods}"
    if [[ ${nPods} == "0" ]]; then
        echo "All pods now scaled down"
        break
    fi
done
date

date
echo "Scaling up all clusters"
SECONDS=0
# Now scale it back up to 3 replicas
for clusterId in ${clusterIds}; do
    echo Scaling up ${clusterId}
    kubectl scale deployment -n kubx-masters master-${clusterId} --replicas=3
    #sleep 1
done

# Wait for all clusters to scale up
declare -i maxCheck=10
declare -i checkCount=${maxCheck}
date
echo "Check all clusters are up and running"
declare -i chkCount
# Start measuring time for all masters to Running after last scale command

# Should check all cluster-health pods are Running before checking master pods
checkClusterHealth=true
while true; do
    nPods=$(kubectl get pod -n kubx-masters | grep "5/5" | grep Running | wc -l)
    echo "${secretID}- Number of Running master pods: ${nPods}"
    if [[ ${checkClusterHealth} == true ]]; then
        nCrashPods=$(kubectl get pod -n kubx-masters | grep CrashLoopBackOff | wc -l)
        nErrorPods=$(kubectl get pod -n kubx-masters | grep Error | wc -l)
        echo "  Number of crashLoopBackOff pods (incl cluster-health): ${nCrashPods}"
        echo "  Number of error pods (incl cluster-health): ${nErrorPods}"
        if [[ ${nCrashPods} != "0" || ${nErrorPods} != "0" ]]; then
            continue
        else
            checkClusterHealth=false
            continue
        fi
    fi
    if [[ ${nPods} == ${numMpods} ]]; then
        if [[ ${checkCount} == 0 ]]; then
            echo "All masters restart confirmed completed at $(date +%Y-%m-%d_%T)"
            break
        else
            echo "All masters restarted.  Continue monitoring for ${checkCount} times to be sure"
            checkCount=$((${checkCount} - 1))
        fi
    else
        checkCount=${maxCheck}
        checkClusterHealth=true
    fi
done
duration=${SECONDS}

nMastersRestarted=$(kubectl get pods -n kubx-masters -o wide | grep master- | grep "5/5" | tr -s ' ' | grep -v "Running 0 " | wc -l)
if [[ ${nMastersRestarted} == 0 ]]; then
    echo "${secretID}: All masters restarted the first time"
else
    kubectl get pods -n kubx-masters -o wide | grep master- | grep "5/5" | tr -s ' ' | grep -v "Running 0 "
    echo "${secretID} DEKs:  ${nMastersRestarted} masters restarted more than once with CrashLoopBackOff"
fi

echo ${secretID}: Time to scale up all clusters: "$(($duration / 60)) minutes and $(($duration % 60)) seconds."
date
