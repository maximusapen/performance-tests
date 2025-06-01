#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# This script will remove all hosts from a Satellite location and trigger an OS reload or cancellation of them.
# Finally it will delete the location, waiting for any associated cluster to have been deleted.

# Position Parameters
# 1. location_name - name of location to cleanup

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

    echo ${location_id}
}

perf_dir=/performance

poll_interval="--poll-interval 30s"
metrics="--metrics"

location_name=$1
if [[ -z "${location_name}" ]]; then
    printf "ERROR: Satellite location name not specified\n"
    exit 1
fi
echo "Cleaning up location ${location_name}"

# Assume this is a location_name first.  Get location id
location_id=$(get_location_id_from_name ${location_name})
if [[ -z "${location_id}" ]]; then
    # Can't find location_id based on parameter.  It must be location id.  If not. nothing we can do but carry on.
    location_id=${location_name}
fi

echo "location_id: ${location_id}"

# Remove and reload all the hosts attached to the location
printf "\n%s - Removing Satellite cluster hosts from location\n" "$(date +%T)"
${perf_dir}/bin/armada-perf-client2 sat host rm --location ${location_id} --cluster "" --cancel --reload

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

# Remove any provisioned server instances, hosts that have been created but are not attached to the location.
printf "\n%s - Deleting unattached Satellite hosts associated with the location\n" "$(date +%T)"
${perf_dir}/bin/armada-perf-client2 sat host rm --location ${location_id} --provisioned --cancel

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

printf "\n%s - Existing Satellite location should not list ${location_name}\n" "$(date +%T)"
${perf_dir}/bin/armada-perf-client2 sat location ls
