#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2017, 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Do not set -x as we do not want to expose STAGE_GLOBAL_ARMPERF_LD_APIKEY
# Console output is also easier to read without -x when trying to find which
# micro-services is updated

upgCarrier=$1
refCarrier=$2

if [ -z "${refCarrier}" ]; then
    echo "No reference Carrier."
    echo "Check reference Carrier is in reference-carrier.json and refCarrier-key.json"
    if [ -z "${refCarrier}" ]; then
        echo "No update Carrier.  Make sure CARRIER_HOST is selected in job"
    fi
    echo "Usage: getCarrierLDFlags.sh <update Carrier> <reference Carrier>"
    echo "Exiting with error."
    exit 1

fi

if [[ -z "${STAGE_GLOBAL_ARMPERF_LD_APIKEY}" ]]; then
    echo "The STAGE_GLOBAL_ARMPERF_LD_APIKEY is not defined"
    echo "Set STAGE_GLOBAL_ARMPERF_LD_APIKEY from Vault"
    echo "Exiting with error."
    exit 1
fi

LD_API_KEY=${STAGE_GLOBAL_ARMPERF_LD_APIKEY}

projKey=default
envKey=production

# Get the upgrade carrier key
upgCarrierKey=$(cat upgCarrier-key.json | jq -r '."'"${upgCarrier}"'"')

# Check upgrade carrier key exists
if [ ${upgCarrierKey} == null ]; then
    echo "No key found for ${upgCarrier} in upgCarrier-key.json."

    echo "Exiting with error."
    exit 1
fi

# Get the reference carrier key
refCarrierKey=$(cat refCarrier-key.json | jq -r '."'"${refCarrier}"'"')

# Check reference carrier key exists
if [ ${refCarrierKey} == null ]; then
    echo "No key found for ${refCarrier} in refCarrier-key.json."

    echo "Exiting with error."
    exit 1
fi

echo ${upgCarrier} : ${upgCarrierKey}
echo ${refCarrier} : ${refCarrierKey}

# LD API to retrieve all flags for carrier
# https://app.launchdarkly.com/api/v2/users/:projKey/:envKey/:userKey/flags

# get performance carrier flags
curl --write-out %{http_code} -H "Authorization: ${LD_API_KEY}" -H "Content-Type: application/json" --output /tmp/${upgCarrier}.json "https://app.launchdarkly.com/api/v2/users/${projKey}/${envKey}/${upgCarrierKey}/flags"

# Extract micro-services version to <carrier-name>.txt
cat /tmp/${upgCarrier}.json | jq '.items' | jq -r 'to_entries[] | "\(.key), \(.value | ._value)"' | sed "s/,/:/" >/tmp/${upgCarrier}.txt
echo "${upgCarrier} config:"
echo ""
cat /tmp/${upgCarrier}.txt

echo ""

if [ ${refCarrier} != "no-reference" ]; then
    # get reference carrier flags
    curl --write-out %{http_code} -H "Authorization: ${LD_API_KEY}" -H "Content-Type: application/json" --output /tmp/${refCarrier}.json "https://app.launchdarkly.com/api/v2/users/${projKey}/${envKey}/${refCarrierKey}/flags"

    # Extract micro-services version to <carrier-name>.txt
    cat /tmp/$refCarrier.json | jq '.items' | jq -r 'to_entries[] | "\(.key), \(.value | ._value)"' | sed "s/,/:/" >/tmp/$refCarrier.txt
    echo "$refCarrier config:"
    echo ""
    cat /tmp/$refCarrier.txt
fi
