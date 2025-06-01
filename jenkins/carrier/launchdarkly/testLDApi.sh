#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Simple test script to test api connection to LD

LD_API_KEY=< STAGE_GLOBAL_ARMPERF_LD_APIKEY from vault >   # pragma: allowlist secret
carrierLDKey=< Get carrier id from upgCarrier-key.json or refCarrier-key.json >

outputFile="testLD"

curl --write-out %{http_code} -H "Authorization: ${LD_API_KEY}" -H "Content-Type: application/json" --output "${outputFile}.json" https://app.launchdarkly.com/api/v2/users/default/production/${carrierLDKey}/flags
cat "${outputFile}.json" | jq '.items' | jq -r 'to_entries[] | "\(.key), \(.value | ._value)"' | sed "s/,/:/" >"${outputFile}.txt"

echo "LD data in {outputFile}.txt"
