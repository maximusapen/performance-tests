#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020,2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Find Old Config
#
# Script will find config directories on a perf client that do not have an existing cluster
# Usage: findOldConfig.sh [-delete]
# [-delete] = Optional flag - if set delete the old config - otherwise just list them

# Get clusters and append them to the existing list of clusters
function appendClusters() {
    response=$(/performance/bin/armada-perf-client2 cluster ls --json 2>&1 )
    # Validate if we had valid json returned
    echo $response | jq empty
    if [[ $? -ne 0 ]]; then
        message="ERROR in https://alchemy-testing-jenkins.swg-devops.com/job/Armada-Performance/job/Automation/job/IdentifyOldClusters/ : Failed to get a list of clusters in findOldConfig.sh - error was: ${response}"
        echo ${message}
        sendToSlack "${message}"
        # Switch tomls back to the original ones
        coriginal
        exit 1
    fi
    clusters=$(echo ${response} | jq -r '.[].name')

    declare -a tmp_cluster_array
    OIFS=$IFS
    IFS=$'\n' 
    tmpcluster_array=($clusters)
    IFS=$OIFS
    cluster_array+=( "${tmpcluster_array[@]}" )

}
# Send a message to slack
function sendToSlack() {
    slackText="$1"
    # For testing, you can DM yourself by changing channel from #armada-perf-private to @<your slack id>
    slackChannel="#armada-perf-private"
    echo ""
    echo "Sending to slack channel ${slackChannel}"
    echo "${slackText}"
    curl -X POST --data-urlencode "payload={\"channel\": \"${slackChannel}\", \"username\": \"webhookbot\", \"text\": \"${slackText}\", \"icon_emoji\": \":ghost:\"}" https://hooks.slack.com/services/T4LT36D1N/B01KW68CKPD/${STAGE_GLOBAL_ARMPERF_SLACKTOKEN}
}

# Functions to switch the config to to different tomls
function csatellite0() {
    export ARMADA_PERFORMANCE_API_KEY=${STAGE_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}
    ln -fs ${satellite_toml} ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml
}
function cstage() {
    export ARMADA_PERFORMANCE_API_KEY=${STAGE_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}
    cnum=$1
    ln -fs ${armada_perf_dir}/armada-perf-client2/config/carrier${cnum}_stage-perf-metadata.toml ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml
}
function cprod-ussouth() {
    export ARMADA_PERFORMANCE_API_KEY=${PROD_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}
    ln -fs ${prod_toml} ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml
}
function coriginal() {
    ln -fs ${orig_meta_toml} ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml
}

# Check no runAuto processes running, as messing with toml files could impact running tests
pids=$(ps -ef | grep Auto | grep -v grep | grep -v cleanAutoProcesses | grep -v " vi ")
if [[ -n "${pids}" ]]; then
    echo "Found Auto pid running - will not run cleanup: ${pids}"
    exit 1
fi

if [[ $1 == "-delete" ]]; then
    delete=true
    echo "Found '-delete' parameter so will delete any config without corresponding cluster."
else
    delete=false
    echo "Will find any config directories without corresponding cluster. If you want to delete them then call this script with '-delete' argument"
fi

perf_dir=/performance
armada_perf_dir=${perf_dir}/armada-perf
perf_config_dir=${perf_dir}/config
export GOPATH=${perf_dir}
orig_meta_toml=$(readlink -f ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml)
satellite_toml="${armada_perf_dir}/armada-perf-client2/config/satellite0_stage-perf-metadata.toml"
prod_toml="${armada_perf_dir}/armada-perf-client2/config/carrierussouth_prod-perf-metadata.toml"

# Calculate the linked carrier from the perf client name
OIFS=$IFS
IFS="-"
ahost=$(hostname)
# Split HOST into array based on "-" delimiter
env_array=($ahost)
IFS=$OIFS
carrierNum=${env_array[2]: -1}

echo "Carrier for ${ahost} is : ${carrierNum}"

declare -a cluster_array

# Each perf client could have clusters from prod, satellite or its linked carrier
# so gather a list of clusters on all these
if [ -f "${prod_toml}" ]; then
    printf "Listing clusters from Prod Carrier\n" 
    cprod-ussouth
    appendClusters
fi

if [ -f "${satellite_toml}" ]; then
    printf "Listing clusters from Satellite0\n" 
    csatellite0
    appendClusters
fi

# perf0 & perf1 don't have corresponding stage carriers
if [ "${carrierNum}" -ge "2" ]; then
    printf "Listing clusters from stage-carrier-${carrierNum}\n" 
    cstage ${carrierNum}
    appendClusters
fi

if [ "${#cluster_array[@]}" -eq "0" ]; then
    echo "WARNING Found 0 clusters on the carriers - won't delete anything as a safety net"
    delete=false
fi

echo "Found clusters: ${cluster_array[*]}"

for dir in `ls ${perf_config_dir} | grep -vE "carrier|satellite"` 
do
    if [[ -z "${dir}" ]]; then
        echo "WARNING - found unexpected empty dir variable, skipping"
        continue
    fi

    if [[ " ${cluster_array[@]} " =~ " ${dir} " ]]; then
        echo "Found cluster for directory /performance/config/${dir}"
    else
        echo "Did not find cluster for directory /performance/config/${dir} - eligible for delete"
        if [[ $delete == "true" ]]; then
            echo "Deleting /performance/config/${dir}"
            rm -rf ${perf_config_dir}/${dir}
        fi
    fi
done

# Switch tomls back to the original ones
coriginal
