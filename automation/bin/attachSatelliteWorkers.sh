#!/bin/bash -e

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
hostQuantity=$2
if [[ -n "${hostQuantity}" ]]; then
    quantityStr="--quantity ${hostQuantity}"
fi

# Attach the Openshift cluster hosts
printf "\n%s - Attaching Satellite Openshift cluster hosts\n" "$(date +%T)"
iaas_type=${cluster_type#*-}
apc2_with_retry "sat host attach --location ${location_id} ${quantityStr} --infrastructure-type ${iaas_type,,} --operating-system ${operating_system^^} --automate --private-key $HOME/.ssh/id_rsa_armada_perf ${metrics} ${poll_interval}"
