#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# This script will disable Portworx on a VPC Gen2 cluster
#
# Input Parameter:
# 1. Cluster Name
# Also requires the KUBECONFIG & ARMADA_PERFORMANCE_API_KEY env variables to be setup.

waitForResourceDelete() {
    resourceType=$1
    maxWaitTime=$2
    resource_id=$3
    cluster=$4
    worker=$5
    printf "\n%s - Waiting for ${resourceType} ${resource_id} to delete \n" "$(date +%T)"

    curWaitTime=0

    resourceGone=false
    pollingInterval=20
    SECONDS=0
    # Need to set this otherwise it will exit when the resources are not found
    set +e
    while [[ ${curWaitTime} -lt ${maxWaitTime} ]]; do
        case $resourceType in
        "volume")
            # Check volume exists
            resourceCheck=$(ibmcloud is volume ${resource_id} --output JSON)
        ;;
        "attachment")
            # Check attachment is gone
            resourceCheck=$(ibmcloud ks storage attachment get --attachment ${resource_id} --cluster ${cluster_name} --worker ${worker} --json)
        ;;
        *) # Default case
            printf "\n%s - Invalid resourceType specified: %s \n" "$(date +%T)" "${resourceType}"
            exit 1 
        ;;
        esac    

        if [[ -n ${resourceCheck} ]]; then
            sleep ${pollingInterval}
            ((curWaitTime += ${pollingInterval}))
        else
            resourceGone=true
            break
        fi
    done
    set -e

    if [[ "${resourceGone}" != true ]]; then
        printf "\n%s - Gave up waiting for \"%s\" Resource to be deleted. Exiting.\n\n" "$(date +%T)" "${resource_id}"
        exit 1
    fi
    printf "\n%s - Resource deleted in \"%s\" seconds\n" "$(date +%T)" $SECONDS
}

# From instructions at https://cloud.ibm.com/docs/containers?topic=containers-utilities#storage-util-rm-vpc-cli
detachStorageGen2() {
    printf "\n%s - Detaching and deleting storage volumes for VPC Gen2 cluster \n" "$(date +%T)"
    cluster_name=$1
    OIFS=$IFS
    IFS=$'\n'
    # Provision and attach storage to every worker in the cluster
    for worker_name in $(${perf_dir}/bin/armada-perf-client2 worker ls --cluster ${cluster_name} --json | jq -r '.[] | .id'); do
        # Find any attachments that have pwxvol in the volume name
        for attachment_id in $(ibmcloud ks storage attachment ls --cluster ${cluster_name} --worker ${worker_name} --json | jq -r ".volume_attachments[] | select (.volume.name|contains(\"pwxvol\")) | .id"); do
            # Get the attachment so we can get the volume ID
            attachment_data=$(ibmcloud ks storage attachment get --attachment ${attachment_id} --cluster ${cluster_name} --worker ${worker_name} --json)
            printf "\n%s - Removing attachment %s for worker %s \n" "$(date +%T)" "${attachment_id}" "${worker_name}"
            # Remove the attachment
            ibmcloud ks storage attachment rm --attachment ${attachment_id} --cluster ${cluster_name} --worker ${worker_name}
            waitForResourceDelete "attachment" 600 ${attachment_id} ${cluster_name} ${worker_name}
            volume_id=$(echo ${attachment_data} | jq -r '.volume.id')
            
            # Delete the volume
            printf "\n%s - Deleting volume %s \n" "$(date +%T)" "${volume_id}"
            ibmcloud is volume-delete ${volume_id} --force
            waitForResourceDelete "volume" 600 ${volume_id}
        done
    done
    IFS=$OIFS
}

# ARMADA_PERFORMANCE_API_KEY environment must be set
if [[ -z "${ARMADA_PERFORMANCE_API_KEY}" ]]; then
    printf "ARMADA_PERFORMANCE_API_KEY not set. Exiting.\n"
    exit 1
fi

# KUBECONFIG environment must be set
if [[ -z "${KUBECONFIG}" ]]; then
    printf "KUBECONFIG not set. Exiting.\n"
    exit 1
fi
run_on_cluster=$1
if [[ -z "${run_on_cluster}" ]]; then
    printf "Cluster not specified. Exiting.\n"
    exit 1
fi

perf_dir=/performance
helm_dir=/usr/local/bin
PWX_NAMESPACE="kube-system"
PWX_RELEASE="portworx-perf"

# Need to login to IBM Cloud so we can use the ibmcloud is commands
PERF_METADATA_TOML=${perf_dir}/armada-perf/armada-perf-client2/config/perf-metadata.toml
IKS_ENDPOINT=$(/performance/bin/tomlToJson $PERF_METADATA_TOML | jq -r '.iks.endpoint')
API_ENDPOINT=$(/performance/bin/tomlToJson $PERF_METADATA_TOML | jq -r '.ibmcloud.iam_endpoint' | cut -d '.' -f2-)
REGION="us-south"
ibmcloud plugin install container-service -r "IBM Cloud" -f
ibmcloud plugin update container-service -r "IBM Cloud" -f
ibmcloud plugin install vpc-infrastructure -r "IBM Cloud" -f
ibmcloud plugin update vpc-infrastructure -r "IBM Cloud" -f
export IBMCLOUD_API_KEY=${ARMADA_PERFORMANCE_API_KEY}
ibmcloud login -a ${API_ENDPOINT} -r ${REGION}

printf "\n\n${grn}Login into IBM Kubernetes Service${end}\n\n"
ibmcloud ks init --host ${IKS_ENDPOINT}

# Instructions from https://cloud.ibm.com/docs/containers?topic=containers-portworx#remove_portworx
curl  -fSsL https://install.portworx.com/px-wipe | bash -s -- --talismanimage icr.io/ext/portworx/talisman --talismantag 1.1.0 --wiperimage icr.io/ext/portworx/px-node-wiper --wipertag 2.5.0 --force

printf "\n%s - Uninstalling Portworx using Helm \n" "$(date +%T)" 
${helm_dir}/helm uninstall ${PWX_RELEASE} -n ${PWX_NAMESPACE} --timeout 900s
printf "\n%s - Portworx uninstall completed \n" "$(date +%T)" 

set +e
printf "\n%s - Remaining Portworx resources: \n" "$(date +%T)" 
kubectl get storageclasses | grep portworx
kubectl get pods -n kube-system | grep 'portworx\|stork'
set -e

detachStorageGen2 ${run_on_cluster}
