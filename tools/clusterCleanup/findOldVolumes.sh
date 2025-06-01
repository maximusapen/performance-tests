#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Find Old Volumes
#
# Script will find volumes and snapshots in an account that do not have an existing cluster
# Usage: findOldVolumes.sh [-delete]
# [-delete] = Optional flag - if set delete the old volumes and snapshots - otherwise just list them

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

perf_dir=/performance

# Have we been asked to delete volumes?
delete=false
if [[ $1 == "-delete" ]]; then
    delete=true
    echo "Found '-delete' parameter so will delete any volumes without attached instance."
else
    echo "Will find any volumes without attached instance. If you want to delete them then call this script with '-delete' argument"
fi

# Ensure we have an up to date 'is' plugin
ibmcloud plugin install vpc-infrastructure -r "IBM Cloud" -f
ibmcloud plugin update vpc-infrastructure -r "IBM Cloud" -f

# PROD/STAGE_GLOBAL_ARMPERF_IBMCLOUD_APIKEY are injected from Jenkins
apiKeys=(${STAGE_GLOBAL_ARMPERF_IBMCLOUD_APIKEY} ${PROD_GLOBAL_ARMPERF_IBMCLOUD_APIKEY})
apiEndpoints=("https://test.cloud.ibm.com" "https://cloud.ibm.com")
apiEnvironments=("stage" "prod")

# Loop over both of our environments
for i in ${!apiEnvironments[@]}; do

    # Need to login to IBM Cloud so we can use the ibmcloud is commands
    export IBMCLOUD_API_KEY=${apiKeys[$i]}
    ibmcloud login -a ${apiEndpoints[$i]} -r "us-south"

    declare -a pendingVolumes
    declare -a deletedVolumes
    declare -a deletedSnapshots

    # Find all VPC volume IDs
    volumeIDs=($(ibmcloud is volumes | awk NR\>2 | awk '{print $1}'))

    if [[ "${#volumeIDs[@]}" -gt 0 ]]; then
        echo "Checking ${#volumeIDs[@]} volumes..."

        for volumeID in "${volumeIDs[@]}"; do
            # Get the instance reference for this volume to see if it is in use.
            instanceRef=$(ibmcloud is volume $volumeID | grep "Volume Attachment Instance Reference" | awk '{print $5}')

            # Get the status aswell to check for pending volumes
            status=$(ibmcloud is volume $volumeID | grep Status | awk '{print $2}')

            if [[ "${instanceRef}" == "-" ]]; then
                if [[ "${status}" == "pending" ]]; then
                    echo "$volumeID : Pending"
                    pendingVolumes+=("${volumeID}")
                else
                    echo "$volumeID : Reclaimable"
                    if $delete; then
                        # Do a forced delete to avoid the confirmation prompt
                        ibmcloud is volume-delete ${volumeID} -f
                        deletedVolumes+=("${volumeID}")
                        echo "$volumeID : Deleted"
                    fi
                fi
            else
                echo "$volumeID : In use"
            fi
        done
    else
        echo "No VPC volumes to cleanup."
    fi


    # Find any volume snapshots
    snapshotIDs=($(ibmcloud is snapshots | awk NR\>2 | awk '{print $1}'))

    if [[ "${#snapshotIDs[@]}" -gt 0 ]]; then
        echo "Checking ${#snapshotIDs[@]} snapshots..."

        for snapshotID in "${snapshotIDs[@]}"; do
            # Get the source volume that the snapshot was taken from
            sourceVolume=$(ibmcloud is snapshot $snapshotID | grep -A 1 "Source volume" | awk NR\>1 | awk '{print $2}')

            if [[ $sourceVolume == -deleted* ]]; then
                echo "$snapshotID : Reclaimable"
                if $delete; then
                    # Do a forced delete to avoid the confirmation prompt
                    ibmcloud is snapshot-delete ${snapshotID} -f
                    deletedSnapshots+=("${snapshotID}")
                    echo "$snapshotID : Deleted"
                fi
            else
                echo "$snapshotID : In use"
            fi
        done
    else
        echo "No storage snapshots to cleanup."
    fi

    if [ "${#pendingVolumes[@]}" -gt "0" ]; then
        # Send deleted volumes to slack
        slackFile=/tmp/slack.txt
        echo "------------------------------------------------------------------" >${slackFile}
        echo "Summary of pending volumes for ${apiEnvironments[$i]}:" >>${slackFile}
        echo "  (If these persist they may need manual cleanup by the VPC team)" >>${slackFile}
        echo "------------------------------------------------------------------" >>${slackFile}
        echo "\`\`\`" >>${slackFile}
        for volume in ${pendingVolumes[@]} 
        do
            echo ${volume} >>${slackFile}
        done
        echo "\`\`\`" >>${slackFile}

        # Send pending volumes to slack channel #armada-perf-private
        slackText=$(cat ${slackFile})
        sendToSlack "${slackText}"
    fi

    if [[ $delete == "true" ]]; then
        if [ "${#deletedVolumes[@]}" -gt "0" ]; then
            # Send deleted volumes to slack
            slackFile=/tmp/slack.txt
            echo "------------------------------------------------------------------" >${slackFile}
            echo "Summary of deleted volumes with no associated instance for ${apiEnvironments[$i]}:" >>${slackFile}
            echo "------------------------------------------------------------------" >>${slackFile}
            echo "\`\`\`" >>${slackFile}
            for volume in ${deletedVolumes[@]} 
            do
                echo ${volume} >>${slackFile}
            done
            echo "\`\`\`" >>${slackFile}

            # Send deleted volumes to slack channel #armada-perf-private
            slackText=$(cat ${slackFile})
            sendToSlack "${slackText}"
        fi

        if [ "${#deletedSnapshots[@]}" -gt "0" ]; then
            # Send deleted snapshots to slack
            slackFile=/tmp/slack.txt
            echo "------------------------------------------------------------------" >${slackFile}
            echo "Summary of deleted snapshots with no associated volume for ${apiEnvironments[$i]}:" >>${slackFile}
            echo "------------------------------------------------------------------" >>${slackFile}
            echo "\`\`\`" >>${slackFile}
            for volume in ${deletedSnapshots[@]} 
            do
                echo ${volume} >>${slackFile}
            done
            echo "\`\`\`" >>${slackFile}

            # Send deleted snapshots to slack channel #armada-perf-private
            slackText=$(cat ${slackFile})
            sendToSlack "${slackText}"
        fi
    fi
done # Loop over environments