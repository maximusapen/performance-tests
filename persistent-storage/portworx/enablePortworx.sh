#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# This script will enable Portworx on a VPC Gen2 cluster
#
# Input Parameter:
# 1. Cluster Name
# Also requires the KUBECONFIG & ARMADA_PERFORMANCE_API_KEY env variables to be setup.

waitForResource() {
    resourceType=$1
    maxWaitTime=$2
    resource_id=$3
    cluster=$4
    worker=$5
    printf "\n%s - Waiting for ${resourceType} ${resource_id} to create \n" "$(date +%T)"

    curWaitTime=0

    resourceReady=false
    pollingInterval=20
    SECONDS=0
    while [[ ${curWaitTime} -lt ${maxWaitTime} ]]; do
        case $resourceType in
        "volume")
            # Check volume exists and is available
            resourceCheck=$(ibmcloud is volume ${resource_id} --output JSON | jq -r ". |  select (.status==\"available\")")
        ;;
        "attachment")
            # Check attachment exists and is attached
            resourceCheck=$(ibmcloud ks storage attachment get --attachment ${resource_id} --cluster ${cluster} --worker ${worker} --json | jq -r ". | select (.status==\"attached\")")
        ;;
        *) # Default case
            printf "\n%s - Invalid resourceType specified: %s \n" "$(date +%T)" "${resourceType}"
            exit 1 
        ;;
        esac    

        if [[ -z ${resourceCheck} ]]; then
            sleep ${pollingInterval}
            ((curWaitTime += ${pollingInterval}))
        else
            resourceReady=true
            break
        fi
    done

    if [[ "${resourceReady}" != true ]]; then
        printf "\n%s - Gave up waiting for \"%s\" Resource to be available. Exiting.\n\n" "$(date +%T)" "${resource_id}"
        exit 1
    fi
    printf "\n%s - Resource became available in \"%s\" seconds\n" "$(date +%T)" $SECONDS
}

# From instructions at https://cloud.ibm.com/docs/containers?topic=containers-utilities#vpc_cli_attach
attachStorageGen2() {
    printf "\n%s - Provisioning storage volumes for VPC Gen2 cluster \n" "$(date +%T)"
    cluster_name=$1
    OIFS=$IFS
    IFS=$'\n'
    storage_tier="10iops-tier"
    storage_capacity="4000"
    # Provision and attach storage to every worker in the cluster
    for worker_name in $(${perf_dir}/bin/armada-perf-client2 worker ls --cluster ${cluster_name} --json | jq -r '.[] | .id'); do
        zone=$(${perf_dir}/bin/armada-perf-client2 worker get --cluster ${cluster_name} --worker ${worker_name} --json | jq -r '.location')
        vol_name=${worker_name}-pwxvol
        
        printf "\n%s - Creating volume %s for worker %s in zone %s \n" "$(date +%T)" "${vol_name}" "${worker_name}" "${zone}"
        vol_data=$(ibmcloud is volume-create ${vol_name} ${storage_tier} ${zone} --capacity ${storage_capacity} --output JSON)
        vol_id=$(echo ${vol_data} | jq -r '.id')
        waitForResource "volume" 600 ${vol_id}

        # Now attach the volume to the worker node
        attachment_data=$(ibmcloud ks storage attachment create --cluster ${cluster_name} --volume ${vol_id} --worker ${worker_name} --output json)
        attachment_id=$(echo ${attachment_data} | jq -r '.id')
        waitForResource "attachment" 600 ${attachment_id} ${cluster_name} ${worker_name}
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
PWX_VERSION="2.11.4"

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

# Check if Portworx is already installed, and just re-use if it already exists
installedCheck=$(kubectl get ds -n kube-system -o=json | jq -r ".items[] | select (.metadata.name==\"portworx\")")
if [[ -z ${installedCheck} ]]; then
    attachStorageGen2 ${run_on_cluster}

    # Need Pull Secrets adding to portworx namespace
    ${perf_dir}/armada-perf/automation/bin/setupRegistryAccess.sh ${PWX_NAMESPACE} "false" "false"

    ${helm_dir}/helm repo add ibmcommunity https://raw.githubusercontent.com/IBM/charts/master/repo/community/
    ${helm_dir}/helm repo update

    # NOTE: We need to set the changePortRange variable to true here as otherwise
    #       Portworx and Openshift can attempt to run things on the same ports in 
    #       the 9000-9015 range causing possible Portworx pod failures. Setting this
    #       to true shifts Portworx to the 17000+ range avoiding the clash.
    printf "\n%s - Installing Portworx using Helm %s \n" "$(date +%T)"
    ${helm_dir}/helm install ${PWX_RELEASE} ibmcommunity/portworx --set clusterName=${run_on_cluster},internalKVDB=true,enablePVCController=true,changePortRange=true,imageVersion=${PWX_VERSION},envVars="PX_IMAGE=icr.io/ext/portworx/px-enterprise:${PWX_VERSION}" -n ${PWX_NAMESPACE} --wait --timeout 900s
    printf "\n%s - Portworx installation completed, waiting while it initialises %s \n" "$(date +%T)"

    # Sleep to give chance for Portworx cluster to start
    sleep 120
else
    printf "\n%s - Portworx already installed, will re-use it \n" "$(date +%T)"
fi


printf "\n%s - Portworx version info: %s \n" "$(date +%T)"
${helm_dir}/helm ls -n ${PWX_NAMESPACE}

printf "\n%s - Portworx storageclasses: %s \n" "$(date +%T)"
kubectl get storageclasses | grep portworx

printf "\n%s - Portworx pod status: %s \n" "$(date +%T)"
kubectl get pods -n ${PWX_NAMESPACE} -o=wide | grep 'portworx\|stork'
kubectl get pods -n ${PWX_NAMESPACE} -o=wide | grep 'portworx\|stork' | awk -v pattern="Running" '$3 !~ pattern' | awk '{print $1}' | xargs -L1 kubectl describe pod -n ${PWX_NAMESPACE}

printf "\n%s - Portworx status: %s \n" "$(date +%T)"
PX_POD=$(kubectl get pods -l name=portworx -n ${PWX_NAMESPACE} -o jsonpath='{.items[0].metadata.name}')
kubectl exec $PX_POD -n ${PWX_NAMESPACE} -- /opt/pwx/bin/pxctl status
