#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# This script will create an ODF StorageCluster
#
# Input Parameter:
# 1. Cluster Name

waitForAddonReady() {
    printf "\n%s - Waiting for addon to become Ready \n" "$(date +%T)"
    cluster_name=$1
    maxWaitTime=$2
    addon_name=$3

    curWaitTime=0

    addonReady=false
    pollingInterval=60
    SECONDS=0
    while [[ ${curWaitTime} -lt ${maxWaitTime} ]]; do
        addonCheck=$(${perf_dir}/bin/armada-perf-client2 cluster addon ls --cluster "${cluster_name}" --json | jq -r ".[] | select(.name==\"${addon_name}\") | select (.healthStatus!= null) | select (.healthStatus|contains(\"Addon Ready\"))")

        if [[ -z ${addonCheck} ]]; then
            sleep ${pollingInterval}
            ((curWaitTime += ${pollingInterval}))
        else
            addonReady=true
            break
        fi
    done

    if [[ "${addonReady}" != true ]]; then
        printf "\n%s - Gave up waiting for \"%s\" addon to be ready. Exiting.\n\n" "$(date +%T)" "${addon_name}"
        ${perf_dir}/bin/armada-perf-client2 cluster addon ls --cluster "${cluster_name}"
        exit 1
    fi
    printf "\n%s - Addon became Ready in \"%s\" seconds\n" "$(date +%T)" $SECONDS
}

waitForStorageClusterReady() {
    printf "\n%s - Waiting for storagecluster to become Ready \n" "$(date +%T)"
    cluster_name=$1
    maxWaitTime=$2

    curWaitTime=0

    scReady=false
    pollingInterval=60
    SECONDS=0
    while [[ ${curWaitTime} -lt ${maxWaitTime} ]]; do
        scCheck=$(oc get storagecluster -n openshift-storage -o=json | jq -r ". | .items[] | select(.metadata.name==\"${cluster_name}\") | select (.status.phase!= null) | select (.status.phase|contains(\"Ready\"))")

        if [[ -z ${scCheck} ]]; then
            sleep ${pollingInterval}
            ((curWaitTime += ${pollingInterval}))
        else
            scReady=true
            break
        fi
    done

    if [[ "${scReady}" != true ]]; then
        printf "\n%s - Gave up waiting for \"%s\" Storage Cluster to be ready. Exiting.\n\n" "$(date +%T)" "${cluster_name}"
        oc get StorageCluster -A
        oc get pods -o=wide -n openshift-storage
        oc get pods -o=wide -n openshift-storage | awk 'NR!=1 {print}' | awk -v pattern="Running" '$3 !~ pattern' | awk '{print $1}' | xargs -L1 oc describe pod -n openshift-storage
        exit 1
    fi
    printf "\n%s - StorageCluster became Ready in \"%s\" seconds\n" "$(date +%T)" $SECONDS
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

# https://cloud.ibm.com/docs/openshift?topic=openshift-deploy-odf-vpc
# Note we enable the addon, then create the StorageCluster after, rather than creating as part of the enable addon
# To create with the addon enable would require significant changes to apc2
odf_addon_name="openshift-data-foundation"
storage_cluster_name="ocs-storagecluster"

# We need to ensure these are removed as they can cause the StorageCluster to fail
oc adm policy remove-scc-from-group anyuid system:authenticated
oc adm policy remove-scc-from-group hostaccess system:authenticated

sleep 60

# Check if the addon is already enabled.
odfAddon=$(${perf_dir}/bin/armada-perf-client2 cluster addon ls --cluster "${run_on_cluster}" --json | jq -r ".[] | select(.name==\"${odf_addon_name}\")")
if [[ -z ${odfAddon} ]]; then
    # It's not, we'll enable it
    ${perf_dir}/bin/armada-perf-client2 cluster addon enable ${odf_addon_name} --cluster "${run_on_cluster}" --param "odfDeploy=false"
fi

# Wait for it to be ready
waitForAddonReady ${run_on_cluster} 900 ${odf_addon_name}

storageCluster=$(oc get storagecluster -n openshift-storage -o=json | jq -r ". | .items[] | select(.metadata.name==\"${storage_cluster_name}\")")
if [[ -z ${storageCluster} ]]; then
    oc create -f ./odfCluster.yaml
fi
waitForStorageClusterReady ${storage_cluster_name} 1200

oc get csv -n openshift-storage
