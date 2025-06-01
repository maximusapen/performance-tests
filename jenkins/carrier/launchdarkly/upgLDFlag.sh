#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018, 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Upgrade a micro-service based on reference carrier version

# Do not set -x as we do not want to expose STAGE_GLOBAL_ARMPERF_LD_APIKEY
# Console output is also easier to read without -x when trying to find which
# micro-services is updated
if [ $# -lt 2 ]; then
    echo "Usage: ./upgLDFlag.sh <upgrade Carrier> <micro-service> [<reference Carrier>] "
    echo "  To specify reference carrier:"
    echo "      ./upgLDFlag.sh stage-dal10-carrier4 armada-secure stage-dal09carrier0"
    echo "  To use default reference carrier:"
    echo "      ./upgLDFlag.sh stage-dal10-carrier401 armada-secure"
    echo "Run in armada-performance/jenkins/carrier/launchdarkly directory"
    echo
    exit 1
fi

upgCarrier=$1
flag=$2
refCarrier=$3

if [[ -z "${refCarrier}" ]]; then
    echo
    echo "No reference Carrier."
    echo "Checking reference Carrier in refCarrier-key.json."
    refCarrier=$(./getReferenceCarrier.sh ${upgCarrier})
    echo "Reference carrier for ${upgCarrier}: ${refCarrier}"
    # Put refCarrier in /tmp for Jenkins job description
    echo "REFERENCE_CARRIER=${refCarrier}" >>/tmp/JJEnv.txt
    cat /tmp/JJEnv.txt
fi

if [[ -z "${STAGE_GLOBAL_ARMPERF_LD_APIKEY}" ]]; then
    echo
    echo "The STAGE_GLOBAL_ARMPERF_LD_APIKEY is not defined"
    echo "Set STAGE_GLOBAL_ARMPERF_LD_APIKEY from Vault"
    echo
    echo "Exiting with error."
    exit 1
fi

upgCarrierKey=$(cat upgCarrier-key.json | jq -r '."'"${upgCarrier}"'"')
refCarrierKey=$(cat refCarrier-key.json | jq -r '."'"${refCarrier}"'"')

projKey=default
envKey=production

LD_API_KEY=${STAGE_GLOBAL_ARMPERF_LD_APIKEY}

# Check micro-services to be upgraded is in the armada-secure BOM
./getBomKeys.sh ${upgCarrier} /tmp/bomKeys.txt ${flag}

echo "Getting value for ${flag} from ${refCarrier}"
refCarrierFlag="${refCarrier}-${flag}"

# get value for the micro-service from reference carrier
curl --write-out %{http_code} -H "Authorization: $LD_API_KEY" -H "Content-Type: application/json" --output /tmp/${refCarrierFlag}.json "https://app.launchdarkly.com/api/v2/users/${projKey}/${envKey}/${refCarrierKey}/flags/${flag}"

# Extract micro-services level from curl response
cat /tmp/${refCarrierFlag}.json | jq '._value' | sed 's/"//g' >/tmp/${refCarrierFlag}.txt
flagValue=$(cat /tmp/${refCarrierFlag}.txt)

# Add flagValue for Jenkins description
echo "FLAG_VALUE=${flagValue}" >>/tmp/JJEnv.txt
cat /tmp/JJEnv.txt

echo "$refCarrier level for ${flag}: ${flagValue}"

echo "Update ${flag} to ${flagValue} on ${upgCarrier}"

curlRC=$(curl -X PUT --write-out %{http_code} -H "Authorization: $LD_API_KEY" -H "Content-Type: application/json" -d '{"setting":"'"${flagValue}"'"}' "https://app.launchdarkly.com/api/v2/users/${projKey}/${envKey}/${upgCarrierKey}/flags/${flag}")
if [[ ${curlRC} == 204 ]]; then
    # curl command was successful with HTTP 204 No Content success status response code
    # Broadcast micro-service levels with trim version - only listing the deployed micro-service
    ./broadcastFlags.sh ${upgCarrier} "${flag}" "" --trim
    exit 0
else
    # Failed to upgrade, broadcast failure
    ./broadcastFlags.sh ${upgCarrier} "" "${flag}" --trim
    exit 1
fi
