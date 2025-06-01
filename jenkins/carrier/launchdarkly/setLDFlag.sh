#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2017, 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Upgrade a micro-service with the specified version in GIT commit value

# Do not set -x as we do not want to expose STAGE_GLOBAL_ARMPERF_LD_APIKEY
# Console output is also easier to read without -x when trying to find which
# micro-services is updated

echo "Usage: ./setLDFlag.sh <update Carrier> < micro-service > < value >"
echo "Run in armada-performance/jenkins/carrier/launchdarkly directory"

upgCarrier=$1
flag=$2
flagValue=$3

if [[ -z "${upgCarrier}" ]]; then
    echo
    echo "No upgrade Carrier."
    echo "Check upgrade Carrier is in upgCarrier-key.json."
    echo "Make sure carrier is in upgCarrier-key.json."
    echo
    echo "Exiting with error."
    exit 1
fi

if [[ -z "${flag}" ]]; then
    echo
    echo "No micro-service provided."
    echo
    echo "Exiting with error."
    exit 1
fi

if [[ -z "${flagValue}" ]]; then
    echo
    echo "No value provided for ${flag}."
    echo
    echo "Exiting with error."
    exit 1
fi

# Check micro-services to be upgraded is in the armada-secure BOM
./getBomKeys.sh ${upgCarrier} /tmp/bomKeys.txt ${flag}
echo "Carrier services from bom:"
flags=$(cat /tmp/bomKeys.txt)
echo ${flags}

if [[ -z "${STAGE_GLOBAL_ARMPERF_LD_APIKEY}" ]]; then
    echo
    echo "The STAGE_GLOBAL_ARMPERF_LD_APIKEY is not defined"
    echo "Set STAGE_GLOBAL_ARMPERF_LD_APIKEY from Vault"
    echo
    echo "Exiting with error."
    exit 1
fi

upgCarrierKey=$(cat upgCarrier-key.json | jq -r '."'"${upgCarrier}"'"')

projKey=default
envKey=production

LD_API_KEY=${STAGE_GLOBAL_ARMPERF_LD_APIKEY}

echo "Update ${flag} to ${flagValue} on ${upgCarrier}"

curl -X PUT --write-out %{http_code} -H "Authorization: $LD_API_KEY" -H "Content-Type: application/json" -d '{"setting":"'"${flagValue}"'"}' "https://app.launchdarkly.com/api/v2/users/${projKey}/${envKey}/${upgCarrierKey}/flags/${flag}"

# Check the version is upgraded in LD
# getCarrierLDFlags.sh will create a /tmp/${upgCarrier}.txt with current micro-services versions from LD
./getCarrierLDFlags.sh ${upgCarrier} no-reference

# Change e setting so we can fail with comments
set +e
grepFlagValue=${flagValue}
if [[ ${flagValue} == "--" ]]; then
    grepFlagValue="\-\-"
fi
checkFlagValue=$(grep ${flag} /tmp/${upgCarrier}.txt | grep ${grepFlagValue})
set -e
if [[ -z ${checkFlagValue} ]]; then
    echo "Failed to update ${flag} to ${flagValue} on ${upgCarrier}"
    echo "Check ${flag}: ${flagValue} exists in LaunchDarkly https://app.launchdarkly.com/default/production/features/${flag}/targeting"
    echo
    # Broadcast failure
    ./broadcastFlags.sh ${upgCarrier} "" "${flag}" --trim
    echo "Exiting with error."
    exit 1
fi

echo "Updated ${checkFlagValue} on ${upgCarrier}"

# Broadcast micro-service levels with trim version - only listing the deployed micro-service
./broadcastFlags.sh ${upgCarrier} "${flag}" "" --trim
