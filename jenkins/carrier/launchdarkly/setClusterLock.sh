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

echo "Usage: ./cluster-lock.sh <update Carrier> < lock | unlock >"
echo "Run in armada-performance/jenkins/carrier/launchdarkly directory"

upgCarrier=$1
lockMode=$2

if [[ -z "${upgCarrier}" ]]; then
    echo
    echo "No upgrade Carrier."
    echo "Check upgrade Carrier is in upgCarrier-key.json."
    echo "Make sure carrier is in upgCarrier-key.json."
    echo
    echo "Exiting with error."
    exit 1
fi

if [[ -z "${STAGE_GLOBAL_ARMPERF_LD_APIKEY}" ]]; then
    echo
    echo "The STAGE_GLOBAL_ARMPERF_LD_APIKEY is not defined"
    echo "Set STAGE_GLOBAL_ARMPERF_LD_APIKEY from Vault"
    echo
    echo "Exiting with error."
    exit 1
fi

# Check for update mode - lock or unlock
if [[ "lock" == "$lockMode" ]]; then
    toLock=true
    echo "Going to lock $upgCarrier"
elif [[ "unlock" == "$lockMode" ]]; then
    toLock=false
    echo "Going to unlock $upgCarrier"
else
    echo
    echo "Need to specify lock or unlock"
    echo
    echo "Exiting with error."
    exit 1
fi

upgCarrierKey=$(cat upgCarrier-key.json | jq -r '."'"${upgCarrier}"'"')

projKey=default
envKey=production

LD_API_KEY=${STAGE_GLOBAL_ARMPERF_LD_APIKEY}

echo "Update cluster-lock to $toLock"

curl -X PUT --write-out %{http_code} -H "Authorization: $LD_API_KEY" -H "Content-Type: application/json" -d '{"setting":'"$toLock"'}' "https://app.launchdarkly.com/api/v2/users/${projKey}/${envKey}/${upgCarrierKey}/flags/cluster-lock"
