#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Run on carrier master to monitor master restart continuously.
# Process will run forever so need to manually kill process.
#
# If only KP clusters exists on carrier, you can use openvpnserver pods to get the KP cluster list
#clusterIds=$(kubectl get pod -n kubx-masters | grep openvpnserver | sed "s/-/ /g" | awk '{print $2}')

# Expected number of 5/5 master pods if non-KP clusters also exists on carrier

source ../test_conf.sh

declare -i numCluster=${clusterEnd}-${clusterStart}+1
declare -i numKPpods=${numCluster}*3
echo Expected number of KP master pods for ${numCluster} clusters: ${numKPpods}

toBreak=false
if [[ $1 == "--break" ]]; then
    # When KP is enabled, masters are restarted from 4 containers to 5 containers
    # so we can break and stop the process if all 5 containers of all master pods are running
    toBreak=true
fi

startTime=$(date +%Y-%m-%d_%T)
echo Start: ${startTime}
SECONDS=0
status=toRestart

declare -i maxCheck=10
declare -i checkCount=${maxCheck}

while true; do
    echo
    date
    #mpods=$(kubectl get pod -n kubx-masters | grep "/5")
    #echo ${mpods}
    numTotal=$(kubectl get pod -n kubx-masters | grep "/5" | grep -v NAME | wc -l)
    numRunningPods=$(kubectl get pod -n kubx-masters | grep "5/5" | grep Running | wc -l)
    numPendingPods=$(kubectl get pod -n kubx-masters | grep "/5" | grep Pending | wc -l)
    numTerminatingingPods=$(kubectl get pod -n kubx-masters | grep "/5" | grep Terminating | wc -l)
    numContainerCreatingPods=$(kubectl get pod -n kubx-masters | grep "/5" | grep ContainerCreating | wc -l)
    numOtherPods=$(kubectl get pod -n kubx-masters | grep "/5" | grep -v Running | grep -v Pending | grep -v Terminating | grep -v NAME | wc -l)
    echo checkCount: ${checkCount}
    echo Number of Running master pods: ${numRunningPods}
    echo Number of Pending master pods: ${numPendingPods}
    echo Number of Terminating master pods: ${numTerminatingingPods}
    echo Number of ContainerCreating master pods: ${numContainerCreatingPods}
    echo Number of master pods in other status: ${numOtherPods}
    echo Total number of 5 container master pods: ${numTotal}

    # It may take some time for the first pod to restart, check here
    if [[ ${status} == "toRestart" && ${numRunningPods} != ${numKPpods} ]]; then
        echo Master restart detected at $(date +%Y-%m-%d_%T)
        status=restarting
        checkCount=${maxCheck}
    fi

    # If started, detect when all master pods are restarted and in Running state
    if [[ ${numRunningPods} == ${numKPpods} ]] && [[ ${numPendingPods} == 0 ]] && [[ ${numTerminatingingPods} == 0 ]] && [[ ${numOtherPods} == 0 ]]; then
        # Set status to completed and record duration.  We don't break but continue as there can be a period during restart when there are no activies.
        echo Complete masters restart detected at $(date +%Y-%m-%d_%T)
        status=restarted
        duration=${SECONDS}
        # This is not a forever run, check whether we can break here
        if [[ ${toBreak} == true ]]; then
            if [[ ${checkCount} == 0 ]]; then
                echo All masters restart confirmed completed at $(date +%Y-%m-%d_%T)
                break
            else
                echo "Continue monitoring for ${checkCount} times to be sure"
                checkCount=$((${checkCount} - 1))
            fi
        fi
    else
        # Reset status in case if status was set to restarted as there can be a period with no activities.
        if [[ ${status} == "toRestart" ]]; then
            checkCount=$((${checkCount} - 1))
            echo "Continue monitoring for ${checkCount} times as cluster may already have restarted"
        elif [[ ${status} == "restarted" ]]; then
            echo "Detected activities after idle.  Reseting status to restarting"
            status=restarting
            checkCount=${maxCheck}
        fi
    fi

    sleep 60

done

kubectl get pod -n kubx-masters -o wide | grep "/5"

endTime=$(date +%Y-%m-%d_%T)
echo End: ${endTime}

duration=${SECONDS}

echo Start time: ${startTime}
echo End time: ${endTime}
echo Monitoring master for ${numCluster} clusters: "$(($duration / 60)) minutes and $(($duration % 60)) seconds."
