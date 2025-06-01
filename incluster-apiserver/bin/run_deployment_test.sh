#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# The create/delete large number of deployments can put extreme load to apiserver
# for big clusters like 1000-node.

# This is a stripped down version of run_incluster_apiserver.sh and create/delete
# deployments without running the performance test.

#Defaults
registry=""
environment=""
get_namespace="incluster-apiserver-target"
instances=10
replicas=5
duration=300

helm_dir=/usr/local/bin

while test $# -gt 0; do
    case "$1" in
    -h | --help)
        echo "run_deployment_test.sh - runs kubernetes incluster apiserver test"
        echo " "
        echo "run_deployment_test.sh [options]"
        echo " "
        echo "options:"
        echo "-h, --help                        show brief help"
        echo "-i, --instances                   Number of instances to run - each instance has its own deployment & Service Account"
        echo "-r, --replicas                    The number of pod replicas for each instance"
        echo "-m, --get_namespace               The namespace that will be used in the test for the get requests"
        exit 0
        ;;
    -i | --instances)
        shift
        if test $# -gt 0; then
            instances=$1
        else
            echo "instances number not specified"
            exit 1
        fi
        shift
        ;;
    -r | --replicas)
        shift
        if test $# -gt 0; then
            replicas=$1
        else
            echo "replicas number not specified"
            exit 1
        fi
        shift
        ;;
    -m | --get_namespace)
        shift
        if test $# -gt 0; then
            get_namespace=$1
        else
            echo "get_namespace value not specified"
            exit 1
        fi
        shift
        ;;
    *)
        args="${args} "$*
        break
        ;;
    esac
done

declare -i failCount=0
podFile=/tmp/inclusterPod.log
get_all_pods() {
    sleepTime=60
    kubectl get pod --all-namespaces >${podFile}
    while [[ $? != 0 ]]; do
        # If cluster is really bad, this will become a forever loop.  In which case you should manually abort the test.
        # To automate this.  Add code to loop for max retry and fail test.
        echo "Failed to get pod.  Wait for ${sleepTime}s and retry..."
        ((failCount++))
        sleep ${sleepTime}
        kubectl get pod --all-namespaces >${podFile}
    done
}

date
echo "Instances: ${instances}, Replicas: ${replicas}, Get_Namespace: ${get_namespace}"
SEONDS=0

# Each instance will get from a different namespace
date
SECONDS=0
echo Creating incluster-apiserver-target and namespace
declare -a created_namespaces
for ((i = 1; i <= ${instances}; i = i + 1)); do
    ns="${get_namespace}${i}"
    namespaces=$(kubectl get namespaces | grep -w "${ns}" | wc -l)
    if [ "$namespaces" -eq 0 ]; then
        echo "Get_namespace  ${ns} does not exist, will create and populate with pods/secrets"
        kubectl create namespace ${ns}
        helm install "incluster-apiserver-target" --namespace ${ns} ../imageDeploy/incluster-apiserver-target/ --set parameters.getNamespace=${ns} --set replicaCount=${replicas}
    fi
    created_namespaces+=(${ns})
done

declare -i totalPod=${instances}*${replicas}
echo "Waiting for ${totalPod} pods to run"

declare -i getInclusterPod
get_all_pods
inclusterPod=$(cat ${podFile} | grep incluster-apiserver-target | grep Running | wc -l)
printf "%s - Number of incluster pods in Running state: ${inclusterPod}\n" "$(date +%T)"
while [[ ${inclusterPod} -ne ${totalPod} ]]; do
    sleep 60
    get_all_pods
    inclusterPod=$(cat ${podFile} | grep incluster-apiserver-target | grep Running | wc -l)
    printf "%s - Number of incluster pods in Running state: ${inclusterPod}\n" "$(date +%T)"
done

declare -i createDuration=${SECONDS}
date

# Sleep for some time for sysdig agent to react to the new deployment/replicas
sleepTime=180
echo Sleep for ${sleepTime} sec before deleting deployments
sleep ${sleepTime}

# Uninstall the target
date
echo "Uninstalling incluster-apiserver-target and delete namespace"
SECONDS=0
for ns in "${created_namespaces[@]}"; do
    # Delete deployments in parallel
    ./delete_deploy_deployment.sh ${ns} &
done

echo "Waiting for all incluster pods to be deleted"
get_all_pods
inclusterPod=$(cat ${podFile} | grep incluster-apiserver-target | wc -l)
printf "%s - Number of incluster pods: ${inclusterPod}\n" "$(date +%T)"
while [[ ${inclusterPod} -ne 0 ]]; do
    sleep 60
    get_all_pods
    inclusterPod=$(cat ${podFile} | grep incluster-apiserver-target | wc -l)
    printf "%s - Number of incluster pods: ${inclusterPod}\n" "$(date +%T)"
done

declare -i deleteDuration=${SECONDS}

printf "%s - Time taken for Instances: ${instances}, Replicas: ${replicas}:\n" "$(date +%T)"
printf "    Create deployments: $((${createDuration} / 60)) minutes and $((${createDuration} % 60)) seconds.\n"
printf "    Delete deployments: $((${deleteDuration} / 60)) minutes and $((${deleteDuration} % 60)) seconds.\n"
printf "Number of failed apiserver API call: ${failCount}.\n"
