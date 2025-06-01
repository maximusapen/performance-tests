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
# 1. Required. Location Identifier
# 2. Optional. Number of control plane hosts to attach and assign. Default use configuration file.

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
        fi

        sleep 30
        ((counter++))
    done
    set -e
    return 1
}

perf_dir=/performance

poll_interval="--poll-interval 30s --timeout 120m"
metrics="--metrics"
quantityStr=""

if [[ -n $1 ]]; then
    location_id=$1

    hostQuantity=$2
    if [[ -n "${hostQuantity}" ]]; then
        quantityStr="--quantity ${hostQuantity}"
    fi
else
    location_id=${SATELLITE_AUTOMATION_LOCATION_ID}
fi

if [[ -z "${location_id}" ]]; then
    printf "ERROR: Satellite location ID not specified\n"
    exit 1
fi

# First check if the location exists
set +e
location_found=false
location_details=$(${perf_dir}/bin/armada-perf-client2 sat location get --location ${location_id} --json 2>/dev/null)
if [[ $? -eq 0 ]]; then
    location_found=true
fi
set -e

if [[ "${location_found}" != true ]]; then
    printf "ERROR: '%s' - Location not found\n" "${location_id}"
    exit 1
fi

# Attach the control plane hosts
printf "\n%s - Attaching Satellite control plane hosts\n" "$(date +%T)"
iaas_type=${cluster_type#*-}
apc2_with_retry "sat host attach --location ${location_id} ${quantityStr} --infrastructure-type ${iaas_type,,} --operating-system ${operating_system^^} --control --automate --private-key $HOME/.ssh/id_rsa_armada_perf ${metrics} ${poll_interval}"

# Assign the control plane hosts
printf "\n%s - Assigning Satellite control plane hosts\n" "$(date +%T)"
apc2_with_retry "sat host assign --location ${location_id} ${quantityStr} ${metrics} ${poll_interval}"
