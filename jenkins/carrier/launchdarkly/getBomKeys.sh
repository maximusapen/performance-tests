#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

upgCarrier=$1
bomFile=$2
flag=$3

# Use armada-secure bom as reference for micro-service update
upgCarrierBom=$(cat upgCarrier-bom.json | jq -r '."'"${upgCarrier}"'"')
bom=${WORKSPACE}/armada-secure/boms/${upgCarrierBom}
bomKeys=$(cat ${bom} | jq '.carriers.services[].name' | sed 's/"//g')
echo "Micro-services included in ${upgCarrierBom}:"
echo
echo ${bomKeys}
echo
# BOM from armada-secure does not include itself.  Add armada-secure to bomKeys
bomKeys="armada-secure ${bomKeys}"
# Save bomKeys in /tmp for slack broadcast
echo ${bomKeys} >>${bomFile}

echo
echo "****************************************************************************************************************"
echo "${upgCarrier} is using BOM in https://github.ibm.com/alchemy-containers/armada-secure/tree/master/bom/${upgCarrierBom}"

if [[ -n "${flag}" ]] && [[ ${bomKeys} != *${flag}* ]]; then
    echo
    echo "${flag} is not found in the BOM for ${upgCarrier}".
    echo "${flag} will not be deployed to ${upgCarrier}".
    echo "****************************************************************************************************************"
    exit 1
fi

echo "****************************************************************************************************************"
echo
