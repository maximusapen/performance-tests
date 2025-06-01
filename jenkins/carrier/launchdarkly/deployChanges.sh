#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Do not set -x as we do not want to expose STAGE_GLOBAL_ARMPERF_LD_APIKEY
# Console output is also easier to read without -x when trying to find which
# micro-services is updated

# Run getCarrierLDFlags.sh before running this script to generate the LD flags

if [ $# -lt 3 ]; then
    echo "Usage: For query mode:"
    echo "       ./deployChanges.sh <update Carrier> <reference Carrier> < --query | --upgrade | --upgrade-all >"
    echo "       To query which micro-service will be updated, use --query:"
    echo "       ./deployChanges.sh <update Carrier> <reference Carrier> --query"
    echo "       To upgrade, use --upgrade:"
    echo "       ./deployChanges.sh <update Carrier> <reference Carrier> --upgrade"
    echo "       To upgrade all microservices to override LD rules, use --upgrade-all:"
    echo "       ./deployChanges.sh <update Carrier> <reference Carrier> --upgrade-all"
    echo "Run in armada-performance/jenkins/carrier/launchdarkly directory after running getCarrierLDFlags.sh"

    echo "Exiting with error."
    exit 1
fi

upgCarrier=$1
refCarrier=$2

if [[ -z "${refCarrier}" ]]; then
    echo "No reference Carrier."
    echo "Check reference Carrier is in reference-carrier.json if using default reference."
    echo "Make sure carrier is in refCarrier-key.json."
    if [[ -z "${refCarrier}" ]]; then
        echo "No update Carrier.  Make sure CARRIER_HOST is selected if running in Jenkins"
    fi

    echo "Exiting with error."
    exit 1
fi

# Check for instruction mode - UPGRADE or QUERY
if [[ "--upgrade" == "$3" ]]; then
    upgrade=true
    upgradeAll=false
    echo "Running in UPGRADE mode --upgrade"
elif [[ "--upgrade-all" == "$3" ]]; then
    upgrade=true
    upgradeAll=true
    echo "Running in UPGRADE mode --upgrade-all"
elif [[ "--query-all" == "$3" ]]; then
    upgrade=false
    upgradeAll=true
    echo "Running in QUERY UPDATE ALL mode --query-all"
else
    # --upgrade option
    upgrade=false
    upgradeAll=false
    echo "Running in QUERY mode"
fi

if [[ -z "${STAGE_GLOBAL_ARMPERF_LD_APIKEY}" ]]; then
    echo "The STAGE_GLOBAL_ARMPERF_LD_APIKEY is not defined"
    echo "Set STAGE_GLOBAL_ARMPERF_LD_APIKEY from Vault"

    echo "Exiting with error."
    exit 1
fi

./getBomKeys.sh ${upgCarrier} /tmp/bomKeys.txt
bomKeys=$(cat /tmp/bomKeys.txt)
echo "Carrier services from bom:"
echo ${bomKeys}

projKey=default
envKey=production

if [ -f "/tmp/ignore_list.txt" ]; then
    jobIgnoreKeyList=$(cat /tmp/ignore_list.txt)
else
    jobIgnoreKeyList=""
fi
echo jobIgnoreKeyList=${jobIgnoreKeyList}
# Ignore list when processing micro-services.  The cluster-lock is handled separately so always ignored.
ignoreKeyList="cluster-lock"
echo ignoreKeyList=${ignoreKeyList}
echo "Adding jobIgnoreKeyList to ignoreKeyList"
ignoreKeyList="${ignoreKeyList} ${jobIgnoreKeyList}"
echo ignoreKeyList=${ignoreKeyList}

# Need to upgrade these new pipeline deploy before any micro-services
prereqKeyList="armada-secure cluster-updater"
echo prereqKeyList: ${prereqKeyList}

summary=""
newLine=$'\n'
isUpgraded=false
updatedKeys=""
failUpdatedKeys=""

upgrade_with_retry() {
    key=$1
    keyValue=$2

    set +e
    local retries=3
    local counter=1
    declare -i sleepTime=120

    # Support retry of temperamental commands
    until [[ ${counter} -gt ${retries} ]]; do

        curlRC=$(curl -X PUT --write-out %{http_code} -H "Authorization: ${LD_API_KEY}" -H "Content-Type: application/json" -d '{"setting":"'"${keyValue}"'"}' "https://app.launchdarkly.com/api/v2/users/${projKey}/${envKey}/${upgCarrierKey}/flags/${key}")

        if [[ ${curlRC} == 204 ]]; then
            # curl command was successful with HTTP 204 No Content success status response code
            return 0
        fi
        # Sleep to space out successive curl call to prevent "You've exceeded the API rate limit. Try again later."
        printf "%s - %d. Command failed. Sleep for ${sleepTime} sec and retry.\n" "$(date +%T)" "${counter}"
        sleep ${sleepTime}

        # Add sleepTime in case next retry fails
        sleepTime=${sleepTime}+60
        ((counter++))
    done
    set -e
    return 1

}

# function to check key between upg and ref carriers.  Update upgCarrier if key
# values are different for UPGRADE mode and key is not in ignore list.
#
processKey() {
    flagKey=$1
    echo ${flagKey}
    upgKeyValue=$(cat /tmp/${upgCarrier}.json | jq -r '.items."'${flagKey}'"."_value"')
    refKeyValue=$(cat /tmp/${refCarrier}.json | jq -r '.items."'${flagKey}'"."_value"')
    upgKeySetting=$(cat /tmp/${upgCarrier}.json | jq -r '.items."'${flagKey}'"."setting"')

    # Value is the current version that will be deployed to carrier when it is unlocked - not necessarily what is running now.
    # Setting is the value that is pinned for that particular carrier. In general we pin all versions so we can control them.
    echo "    ${upgCarrier} value: ${upgKeyValue}"
    echo "    ${upgCarrier} setting: ${upgKeySetting}"
    echo "    ${refCarrier}: ${refKeyValue}"

    isUpgraded=false
    # If upgKeySetting is null this likely means it is a new Microservice that has been added, so
    # will inherit the setting from reference carrier. We want to process this so that we can set the version
    if [[ ${refKeyValue} != ${upgKeyValue} || ${upgradeAll} == true || ${upgKeySetting} == "null" ]]; then

        LD_API_KEY=${STAGE_GLOBAL_ARMPERF_LD_APIKEY}

        upgradeMsg="Update ${flagKey} from ${upgKeyValue} to ${refKeyValue}"
        summary="$summary$newLine${upgradeMsg}"

        toIgnore=false
        for ignoreKey in ${ignoreKeyList}; do
            if [[ ${flagKey} == ${ignoreKey} ]]; then
                toIgnore=true
            fi
        done

        if [[ ${toIgnore} == true ]]; then
            ignoreMsg="    ${flagKey} ignored"
            echo $ignoreMsg
            summary="$summary$newLine$ignoreMsg"
        elif [[ $refKeyValue == "--" || $refKeyValue == "null" ]]; then

            # If someone set the LD flag to default "--", we don't update as we
            # need to preserve the version that is being deployed on our carrier
            # Update manually with upgrade-microservice job if really want "--"

            ignoreMsg="    $flagKey with value -- ignored.  Use upgrade-microservice job for -- value."
            echo $ignoreMsg
            summary="$summary$newLine$ignoreMsg"

        else
            if [[ ${upgrade} == true ]]; then
                if [[ ${currentLockValue} == "true" ]]; then
                    echo "Carrier is locked.  Update cluster-lock from ${currentLockValue} to false"
                    if [[ ${upgrade} == true ]]; then
                        ./setClusterLock.sh ${upgCarrier} unlock
                        currentLockValue="false"
                        echo "Sleep for some time for carrier to unlock"
                        sleep 60
                    fi
                fi

                upgCarrierKey=$(cat upgCarrier-key.json | jq -r '."'"${upgCarrier}"'"')
                upgrade_with_retry ${flagKey} ${refKeyValue}
                if [[ $? -eq 0 ]]; then
                    isUpgraded=true
                    updatedKeys="${updatedKeys} ${flagKey}"
                else
                    failUpdatedKeys="${failUpdatedKeys} ${flagKey}"
                fi
                # Sleep to space out successive curl call to prevent "You've exceeded the API rate limit. Try again later."
                sleep 10
            fi
        fi
    fi
}

echo
echo Handle cluster-lock - unlock if locked
echo

origClusterLockValue=$(cat /tmp/${upgCarrier}.json | jq -r '.items."cluster-lock"."_value"')
echo "cluster-lock before job: ${origClusterLockValue}"
currentLockValue=${origClusterLockValue}

echo
echo Process armada-secure before cluster-updater
echo
if [[ ${bomKeys} == *armada-secure* ]]; then
    # armada-secure in bom
    processKey armada-secure
    if [[ ${isUpgraded} == true ]]; then
        echo "armada-secure is upgraded, sleep for 60s"
        sleep 60
    fi
fi

echo
echo Process cluster-updater
echo
if [[ ${bomKeys} == *cluster-updater* ]]; then
    processKey cluster-updater
    if [[ ${isUpgraded} == true ]]; then
        echo "cluster-updater is upgraded, sleep for 120s"
        sleep 120
    fi
fi

# Upgrade the micro-services to upgCarrier
echo
echo Processing micro-services now
echo

sleepTime=0
for flagKey in ${bomKeys}; do
    if [[ ${prereqKeyList} == *${flagKey}* ]]; then
        echo "    ${flagKey} already processed"
    else
        processKey ${flagKey}
    fi
done

# Print summary before sleep
echo "+++++++++++++++++++++++++++++++++"
echo "Upgrade summary: (no change if nothing listed below)"
echo "upgrade: ${upgrade}  upgradeAll: ${upgradeAll}"
echo "$summary"
echo "+++++++++++++++++++++++++++++++++"

# Broadcast micro-service levels if carrier has been upgraded
if [[ ${updatedKeys} != "" || ${failUpdatedKeys} != "" ]]; then
    echo "Broadcasting these keys:"
    echo "  updatedKeys: ${updatedKeys}"
    echo "  failUpdatedKeys: ${failUpdatedKeys}"
    ./broadcastFlags.sh ${upgCarrier} "${updatedKeys}" "${failUpdatedKeys}"
    if [[ ${updatedKeys} != "" ]]; then
        echo "updatedKeys=\"${updatedKeys}\"" >/tmp/updatedKeys.log
        echo "/tmp/updatedKeys.log created for mark-cruiser-bom-release Jenkins job"
    fi
fi

# Lock the carrier if carrier was originally locked
if [[ ${origClusterLockValue} == "true" ]] && [[ ${currentLockValue} == "false" ]]; then
    echo "Lock carrier after update as carrier was originally locked"
    if [[ ${upgrade} == true ]]; then
        # Big carriers upgraded weekly can take a long time to upgrade.  Long sleep to be sure.
        echo "Sleep 10 min for micro-services to be deployed before locking carrier"
        sleep 600
        echo "Now locking carrier"
        ./setClusterLock.sh ${upgCarrier} lock
    fi
fi
