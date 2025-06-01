#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020,2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Delete Old clusters
#
# Script will just find clusters older than 6 days unless 'true' is passed in as the first parameter
# Note - it will only find clusters that have > 0 workers as cluster without workers are not an issue.
# This will run against multiple carriers.
# Usage: findOldClusters.sh true|false [clusterid1,clusterid2]
# true|false : if true delete the old clusters - if false just list them
# [clusterid1,clusterid2] : optional list of comma separated clusters that are exempt from deletions

# Send message to slack
function sendToSlack() {
    slackText="$1"
    # For testing, you can DM yourself by changing channel from #armada-perf-private to @<your slack id>
    slackChannel="#armada-perf-private"
    echo ""
    echo "Sending to slack channel ${slackChannel}"
    echo "${slackText}"
    curl -X POST --data-urlencode "payload={\"channel\": \"${slackChannel}\", \"username\": \"webhookbot\", \"text\": \"${slackText}\", \"icon_emoji\": \":ghost:\"}" https://hooks.slack.com/services/T4LT36D1N/B01KW68CKPD/${STAGE_GLOBAL_ARMPERF_SLACKTOKEN}
}
# Function to restore the original toml file
function coriginal() {
    ln -fs ${orig_meta_toml} ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml
}

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

        /performance/bin/armada-perf-client2 ${apc2_command}

        if [[ $? == 0 ]]; then
            # Command was successful
            return 0
        fi

        sleep 30
        ((counter++))
    done
    # If we get to here the retries have all failed, so send slack notification
    message="ERROR in https://alchemy-testing-jenkins.swg-devops.com/job/Armada-Performance/job/Automation/job/IdentifyOldClusters/ : Failed to run apc2 command ${apc2_command}"
    echo ${message}
    sendToSlack "${message}"
}

# Check no runAuto processes running, as messing with toml files could impact running tests
pids=$(ps -ef | grep Auto | grep -v grep | grep -v cleanAutoProcesses | grep -v " vi ")
if [[ -n "${pids}" ]]; then
    echo "Found Auto pid running - will not run cleanup: ${pids}"
    exit 1
fi

if [[ $1 == "true" ]]; then
    delete=true
    echo "Delete option specified, so will remove any clusters older than 6 days."
elif [[ $1 == "false" ]]; then
    delete=false
    echo "Will find clusters older than 6 days. If you want to delete them then call this script with the first parameter set to true"
else
    echo "First parameter must be true|false to control if the clusters will actually be deleted."
    exit 1
fi
exemptions=$2

echo "The following clusters are exempt from deletion: ${exemptions}"

perf_dir=/performance
armada_perf_dir=${perf_dir}/armada-perf
export GOPATH=${perf_dir}

# The number of days to consider a cluster too old
age_to_delete=6

# Copy current perf-metadata.toml location so that we can restore later
orig_meta_toml=$(readlink -f ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml)

# Get list of clusters in each carrier
echo "Retrieving a list of clusters in each carrier"
declare -a old_clusters
declare -a deleted_clusters
declare -a exempt_clusters

# Iterate over all our stage carriers plus prod & satellite
for carrierNum in 4 5 prod sat; do
    printf "Listing clusters from Carrier %s \n" "${carrierNum}"
    if [[ "${carrierNum}" == "prod" ]]; then
        export ARMADA_PERFORMANCE_API_KEY=${PROD_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}
	    ln -fs ${armada_perf_dir}/armada-perf-client2/config/carrierussouth_prod-perf-metadata.toml ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml
    elif [[ "${carrierNum}" == "sat" ]]; then
        export ARMADA_PERFORMANCE_API_KEY=${STAGE_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}
        ln -fs ${armada_perf_dir}/armada-perf-client2/config/satellite0_stage-perf-metadata.toml ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml
    else
        export ARMADA_PERFORMANCE_API_KEY=${STAGE_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}
        ln -fs ${armada_perf_dir}/armada-perf-client2/config/carrier${carrierNum}_stage-perf-metadata.toml ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml
    fi

    clusters=$(/performance/bin/armada-perf-client2 cluster ls --json 2>&1)
    # Validate if we had valid json returned
    echo $clusters | jq empty
    if [[ $? -ne 0 ]]; then
        message="ERROR in https://alchemy-testing-jenkins.swg-devops.com/job/Armada-Performance/job/Automation/job/IdentifyOldClusters/ : Failed to get a list of clusters on carrier${carrierNum} in findOldClusters.sh - error was: ${clusters}"
        coriginal
        echo ${message}
        sendToSlack "${message}"
    fi

    while read clstr; do
        workers=$(echo ${clstr} | jq -r ' .workerCount')
        name=$(echo ${clstr} | jq -r ' .name')
        id=$(echo ${clstr} | jq -r ' .id')
        provider=$(echo ${clstr} | jq -r ' .provider')
        location=$(echo ${clstr} | jq -r ' .location')
        carrier_name_and_id="carrier${carrierNum}_${name}_${id}"
        if [ "$workers" -gt "0" ]; then
            created_date=$(echo ${clstr} | jq -r ' .createdDate')
            created_plus_age=$(date -d "${created_date} ${age_to_delete} days" +%s)
            date_now=$(date +%s)
            if [ $date_now -ge $created_plus_age ]; then
                    echo "${carrier_name_and_id} was created more than ${age_to_delete} days ago (${created_date}) - eligible for delete"
                    is_exempt=$(echo ${exemptions} | grep ${id} | wc -l)
                    if [ ${is_exempt} -eq 0 ]; then
                        if [[ $delete == "true" ]]; then
                            echo "Deleting cluster: "
                            apc2_with_retry "cluster rm --cluster $name --force-delete-storage"
                            deleted_clusters+=("${carrier_name_and_id}_${created_date}")
                            if [[ $provider == "satellite" ]]; then
                                # Do Satellite cleanup
                                apc2_with_retry "sat host rm --location ${location} --cluster ${name} --cancel --reload"
                            fi
                        fi
                        old_clusters+=("${carrier_name_and_id}_${created_date}")
                    else 
                        exempt_clusters+=("${carrier_name_and_id}_${created_date}")
                    fi        
            else
                    # Cluster is not older than ${age_to_delete} days - can ignore
                    continue
            fi
        else
            # Ignore clusters with 0 workers
            continue
        fi
    done < <(echo ${clusters} | jq -c '.[]')
done

# Restore original perf.toml
coriginal

echo "--------------------------------------------------------------------------------" 
echo "Summary of clusters eligible for deletion as older than ${age_to_delete} days:"
echo "--------------------------------------------------------------------------------"
for cluster in ${old_clusters[@]} 
do
    echo ${cluster}
done

if [[ $delete == "true" ]]; then
    if [ "${#deleted_clusters[@]}" -gt "0" ]; then
        # Send deleted clusters to slack
        slackFile=/tmp/slack.txt
        echo "------------------------------------------------------------------" >${slackFile}
        echo "Summary of clusters Deleted as older than ${age_to_delete} days:" >>${slackFile}
        echo "------------------------------------------------------------------" >>${slackFile}
        echo "\`\`\`" >>${slackFile}
        for cluster in ${deleted_clusters[@]} 
        do
            echo ${cluster} >>${slackFile}
        done
        echo "\`\`\`" >>${slackFile}

        # Send deleted clusters to slack channel #armada-perf-private
        slackText=$(cat ${slackFile})
        sendToSlack "${slackText}"
    fi
else
    if [ "${#old_clusters[@]}" -gt "0" ]; then
        # Send deleted clusters to slack
        slackFile=/tmp/slack.txt
        echo "------------------------------------------------------------------" >${slackFile}
        echo "Summary of clusters older than ${age_to_delete} days:" >>${slackFile}
        echo "Please delete if no longer required" >>${slackFile}
        echo "------------------------------------------------------------------" >>${slackFile}
        echo "\`\`\`" >>${slackFile}
        for cluster in ${old_clusters[@]} 
        do
            echo ${cluster} >>${slackFile}
        done
        echo "\`\`\`" >>${slackFile}

        # Send deleted clusters to slack channel #armada-perf-private
        slackText=$(cat ${slackFile})
        sendToSlack "${slackText}"
    fi
fi
if [ "${#exempt_clusters[@]}" -gt "0" ]; then
        # Send deleted clusters to slack
        slackFile=/tmp/slack.txt
        echo "------------------------------------------------------------------" >${slackFile}
        echo "Summary of clusters older than ${age_to_delete} days, but are exempt from deletion:" >>${slackFile}
        echo "Please delete if no longer required" >>${slackFile}
        echo "------------------------------------------------------------------" >>${slackFile}
        echo "\`\`\`" >>${slackFile}
        for cluster in ${exempt_clusters[@]} 
        do
            echo ${cluster} >>${slackFile}
        done
        echo "\`\`\`" >>${slackFile}

        # Send deleted clusters to slack channel #armada-perf-private
        slackText=$(cat ${slackFile})
        sendToSlack "${slackText}"
fi
