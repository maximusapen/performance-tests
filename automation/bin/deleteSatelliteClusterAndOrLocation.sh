#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# This script will make changes to a location based on action:
# --delete-cluster: Delete cluster from location only
# --delete-location:  Delete cluster and location
#
# Script will remove hosts from a Satellite cluster and/or location and trigger an OS reload or cancellation of them.

# Position Parameters
# 1. location_name - name of location.
# 2. cluster_name - name of the cluster to delete.  Need to set to "" if deleting a location without a cluster
# 3. action - --delete-location or --delete-cluster

get_location_id_from_name() {
    local location_name=$1

    set +e
    location_found=false

    # First of all try the obvious approach, get the location by name
    location_details=$(${perf_dir}/bin/armada-perf-client2 sat location get --location ${location_name} --json 2>/dev/null)
    if [[ $? -ne 0 ]]; then
        # We shouldn't have to, but try an alternative approach just in case. - see https://github.ibm.com/alchemy-containers/satellite-planning/issues/1337
        location_details=$(${perf_dir}/bin/armada-perf-client2 sat location ls --json | jq ".[] | select(.name==\"${location_name}\")") 2>/dev/null
    fi

    location_id=$(echo ${location_details} | jq -j '.id')
    set -e

    # Return location id
    echo ${location_id}
}

if [[ ${CLOUD_ENVIRONMENT} == "Stage" ]]; then
    export ARMADA_PERFORMANCE_API_KEY=${STAGE_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}
    export ARMADA_PERFORMANCE_INFRA_API_KEY=${PROD_GLOBAL_ARMPERF_IBMCLOUD_APIKEY} # Use production VPC Iaas for VPC based Satellite tests
elif [[ ${CLOUD_ENVIRONMENT} == "Production" ]]; then
    export ARMADA_PERFORMANCE_API_KEY=${PROD_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}
    export ARMADA_PERFORMANCE_INFRA_API_KEY=${PROD_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}
else
    echo "Unknown CLOUD_ENVIRONMENT ${CLOUD_ENVIRONMENT}.  Please set CLOUD_ENVIRONMENT to Stage or Production."
    exit 1
fi

perf_dir=/performance
export GOPATH=${perf_dir}
export METRICS_DB_KEY="${armada_performance_db_password}"
export METRICS_ROOT_OVERRIDE="${armada_performance_metrics_root_override}"
export ARMADA_PERFORMANCE_CLASSIC_INFRA_API_KEY=${armada_performance_classic_infra_key}

poll_interval="--poll-interval 30s"
metrics="--metrics"

location_name=$1
cluster=$2
action=$3

if [[ -z "${location_name}" ]]; then
    printf "ERROR: Satellite location name not specified\n"
    exit 1
fi
echo "Deleting cluster ${cluster} from location ${location_name}"

# Assume this is a location_name first.  Get location id
location_id=$(get_location_id_from_name ${location_name})

if [[ -z "${location_id}" ]]; then
    # Can't find location_id based on parameter.  It must be location id.  If not. nothing we can do but carry on.
    location_id=${location_name}
fi

echo "Found location_id: ${location_id} from location_name: ${locatiion_name}"

printf "\n%s - Syncing location on client\n" "$(date +%T)"
${perf_dir}/bin/armada-perf-client2 sat location sync --location ${location_name}

# Delete the satellite cluster if specified
if [[ ${cluster} != "" ]]; then
    printf "\n%s - Removing Satellite cluster from location\n" "$(date +%T)"
    ${perf_dir}/bin/armada-perf-client2 cluster rm --cluster ${cluster} --force-delete-storage
fi

# Remove and reload all the hosts attached to the cluster
printf "\n%s - Removing Satellite cluster hosts from location\n" "$(date +%T)"
${perf_dir}/bin/armada-perf-client2 sat host rm --location ${location_id} --cluster ${cluster} --cancel --reload

if [[ ${action} == "--delete-location" ]]; then
    printf "\n%s - Removing Satellite control plane hosts from location\n" "$(date +%T)"
    ${perf_dir}/bin/armada-perf-client2 sat host rm --location ${location_id} --control --cancel --reload

    # Wait for hosts to be removed
    while
        host_count=($(${perf_dir}/bin/armada-perf-client2 sat host ls --location ${location_id} --json | jq -r '.[] | .id ' | wc -l))
        ((host_count > 0))
    do
        printf "%s - Waiting for location hosts to be removed. Hosts remaining '%s'\n" "$(date +%T)" "${host_count}"
        sleep 30
    done

    echo

    # Wait for clusters to be deleted on the location
    while
        sleep 60
        loc_cluster=$(${perf_dir}/bin/armada-perf-client2 cluster ls --provider satellite --json | jq ".[] | select(.location==\"${location_name}\") | .id")
        [[ -n ${loc_cluster} ]]
    do
        printf "%s - Waiting for cluster(s) associated with location '%s' to be deleted\n" "$(date +%T)" ${location_name}
    done

    # Delete the location
    printf "\n%s - Deleting Satellite location\n" "$(date +%T)"
    ${perf_dir}/bin/armada-perf-client2 sat location rm --location ${location_id} ${metrics} ${poll_interval}

    # Finally wait for location to be deleted
    while
        sleep 60
        loc_id=$(${perf_dir}/bin/armada-perf-client2 sat location ls --json | jq ".[] | select(.name==\"${location_name}\") | .id")
        [[ -n ${loc_id} ]]
    do
        printf "%s - Waiting for location '%s' to be deleted\n" "$(date +%T)" ${location_name}
    done

fi

printf "\n%s - Satellite location:\n" "$(date +%T)"
${perf_dir}/bin/armada-perf-client2 sat location ls

printf "\n%s - Satellite clusters:\n" "$(date +%T)"
${perf_dir}/bin/armada-perf-client2 cluster ls
