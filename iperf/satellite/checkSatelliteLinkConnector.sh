#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# Check satellite-link-connector status

# Get IAM token.  Should avoid getting IAM token every time running this script

# Set up an alias and only run gettoken when token expires
# alias gettoken='export IAMTOKEN=$(ibmcloud iam oauth-tokens | awk '\''{print $4}'\'')'

if [[ -z ${IAMTOKEN} ]]; then
    echo "IAMTOKEN missing.  Set up gettoken alias and then run command gettoken"
    exit 1
fi

source envFile

SATLINK_API="https://api.link.satellite.test.cloud.ibm.com"

echo ${SATLINK_API}/v1/locations/$location_id
curl -sS -H "accept: application/json" -H "Content-Type: application/json" -H "Authorization: Bearer ${IAMTOKEN}" ${SATLINK_API}/v1/locations/${location_id} | jq
