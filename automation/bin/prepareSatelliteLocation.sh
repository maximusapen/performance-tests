#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# Position Parameters
# 1. Required. Name of location to create
# 2. Optional. Number of hosts override. Default use all.

# Runs armada-perf-client2 commands with built in retry
apc2_with_retry() {
    apc2_command=$1

    set +e
    local retries=3
    local counter=1

    # Support retry of temperamental commands
    until [[ ${counter} -gt ${retries} ]]; do
        if [[ ${counter} -gt 1 ]]; then
            printf "%s - %d. Command failed. Retrying\n" "$(date +%T)" "${counter}"
        fi

        ${perf_dir}/bin/armada-perf-client2 ${apc2_command}

        if [[ $? == 0 ]]; then
            # Command was successful
            return 0
        else
            # Try to spot the difference between a transient failure and one where the location has gone to deploy_failed
            sleep 30
            location_details=$(get_location_details_from_name ${location_name})
            location_state=$(echo ${location_details} | jq -j '.state')
            if [[ -n ${location_state} ]]; then
                if [[ "${location_state}" == normal ]]; then
                    # This is a little unexpected as the create location returned a failure - but the location is normal so we can carry on
                    return 0
                else
                    # Location is in an error state, so we want to fail
                    printf "WARNING: Satellite location '%s' creation failed, and is now in state '%s' \n" "${location_name}" "${location_state}"
                    exit 1
                fi
            fi
        fi

        sleep 30
        ((counter++))
    done
    set -e
    return 1
}

get_location_details_from_name() {
    local location_name=$1

    set +e
    # First of all try the obvious approach, get the location by name
    location_details=$(${perf_dir}/bin/armada-perf-client2 sat location get --location ${location_name} --json 2>/dev/null)
    if [[ $? -ne 0 ]]; then
        # We shouldn't have to, but try an alternative approach just in case. - see https://github.ibm.com/alchemy-containers/satellite-planning/issues/1337
        location_details=$(${perf_dir}/bin/armada-perf-client2 sat location ls --json | jq ".[] | select(.name==\"${location_name}\")") 2>/dev/null
    fi
    set -e

    echo ${location_details}
}

perf_dir=/performance
armada_perf_dir=${perf_dir}/armada-perf

poll_interval="--poll-interval 30s --timeout 120m"
metrics="--metrics"

location_name=$1
if [[ -z "${location_name}" ]]; then
    printf "ERROR: Satellite location name not specified\n"
    exit 1
fi

automation_flag=$2
if [[ -z "${automation_flag}" ]]; then
    printf "WARNING: Satellite automation flag not specified. Assuming manual run\n"
    automation_flag=false
fi

prepare_for_delete_flag=$3
if [[ -z "${prepare_for_delete_flag}" ]]; then
    printf "WARNING: Prepare for delete flag not specified. Assuming prepare location for testing\n"
    prepare_for_delete_flag=false
fi

# Check for existing location with the speciifed name
location_details=$(get_location_details_from_name ${location_name})
location_id=$(echo ${location_details} | jq -j '.id')

if [[ -n "${location_id}" ]]; then
    # For automated runs, we'll always cleanup and delete the existing location
    if [[ "${automation_flag}" == true ]]; then
        printf "Automation run. Performing cleanup of location '%s'\n" ${location_name}
        source ${armada_perf_dir}/automation/bin/cleanupSatelliteLocation.sh ${location_name}
        printf "\n%s - After location is deleted, cleanup is still going on.  Sleep for 5 mins before creating location of same name.\n" "$(date +%T)"
        sleep 300
    else
        # Manual run
        export SATELLITE_AUTOMATION_LOCATION_ID=${location_id}

        location_normal=$(echo ${location_details} | jq -j 'select(.state=="normal")')
        if [[ -n ${location_normal} ]]; then
            # It's in 'normal' state, all is good.
            printf "Healthy satellite location '%s' already exists\n" ${location_name}
            return 0
        else
            if [[ ${prepare_for_delete_flag} == true ]]; then
                # Location is unhealthy.  Prepare SATELLITE_AUTOMATION_LOCATION_ID for delete location.
                location_details=$(get_location_details_from_name ${location_name})
                location_id=$(echo ${location_details} | jq -j '.id')
                export SATELLITE_AUTOMATION_LOCATION_ID=${location_id}
            else
                # Specified location exists, but is unhealthy. Warn user and return error.
                printf "WARNING: Unhealthy satellite location '%s' already exists\n" ${location_name}
                exit 1
            fi
        fi
    fi
fi

# Create the location
IFS="_" read -ra OS <<<"${operating_system}"
if [[ ${OS[0]} == "RHCOS" ]]; then
    coreos="--coreos-enabled=true"
else
    coreos="--coreos-enabled=false"
fi
printf "\n%s - Creating Satellite location '%s'\n" "$(date +%T)" ${location_name}
apc2_with_retry "sat location create --name ${location_name} ${coreos} ${metrics} ${poll_interval}"

location_details=$(get_location_details_from_name ${location_name})
location_id=$(echo ${location_details} | jq -j '.id')
export SATELLITE_AUTOMATION_LOCATION_ID=${location_id}

printf "\n%s - Setting up Satellite location '%s' control plane\n" "$(date +%T)" "${location_name}"
source ${armada_perf_dir}/automation/bin/prepareSatelliteControlPlane.sh ${location_id}
