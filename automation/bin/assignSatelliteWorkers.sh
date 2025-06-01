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
# 1. Satellite location identiifer
# 2. Name of Openshift cluster to assign hosts
# 3. [Optional] Number of hosts to assign

# Runs armada-perf-client2 commands with built in retry
apc2_with_retry() {
    apc2_command=$1

    set +e
    local retries=3
    local counter=1

    # Support retry of temperamental commands
    until [[ ${counter} -gt ${retries} ]]; do
        if [[ ${counter} -gt 1 ]]; then
            printf "%s - %d. Command failed. Retrying.\n" "$(date +%T)" "${counter}"
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

location_id=$1
cluster_name=$2
if [[ -z "${cluster_name}" ]]; then
    printf "Openshift cluster name not specified\n"
    exit 1
fi
hostQuantity=$3
if [[ -n "${hostQuantity}" ]]; then
    quantityStr="--quantity ${hostQuantity}"
fi

# Assign the Openshift cluster hosts
printf "\n%s - Assigning Satellite Openshift cluster hosts\n" "$(date +%T)"
apc2_with_retry "sat host assign --location ${location_id} --cluster ${cluster_name} ${quantityStr} ${metrics} ${poll_interval}"
