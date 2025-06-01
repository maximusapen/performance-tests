#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# This script will Delete ODF StorageCluster and disable ODF adon
#
# Input Parameter:
# 1. Cluster Name

waitForStorageClusterDeleted() {
    printf "\n%s - Waiting for storagecluster to be deleted \n" "$(date +%T)"
    cluster_name=$1
    maxWaitTime=$2

    curWaitTime=0

    scGone=false
    pollingInterval=60
    SECONDS=0
    while [[ ${curWaitTime} -lt ${maxWaitTime} ]]; do
        scCheck=$(oc get storagecluster -n openshift-storage -o=json | jq -r ". | .items[] | select(.metadata.name==\"${cluster_name}\")")

        if [[ -n ${scCheck} ]]; then
            sleep ${pollingInterval}
            ((curWaitTime += ${pollingInterval}))
        else
            scGone=true
            break
        fi
    done

    if [[ "${scGone}" != true ]]; then
        printf "\n%s - Gave up waiting for \"%s\" Storage Cluster to be Deleted - will carry on with cleanup anyway \n" "$(date +%T)" "${cluster_name}"
        oc get storagecluster -A
    else
        printf "\n%s - StorageCluster was deleted in \"%s\" seconds\n" "$(date +%T)" $SECONDS
    fi
}

perf_dir=/performance

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

# https://cloud.ibm.com/docs/openshift?topic=openshift-ocs-manage-deployment#ocs-addon-rm
odf_addon_name="openshift-data-foundation"
ocs_cluster_name="ocs-storagecluster"

oc delete ocscluster ${ocs_cluster_name} --wait=false --ignore-not-found=true

waitForStorageClusterDeleted ${ocs_cluster_name} 600

#Cleanup extra resources
. ./cleanup.sh

${perf_dir}/bin/armada-perf-client2 cluster addon disable ${odf_addon_name} --cluster "${run_on_cluster}"