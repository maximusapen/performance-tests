#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# To be used by all other scripts to return the default reference carrier for a performance carrier

if [ $# -lt 1 ]; then
    echo "No upgrade carrier."
    echo "Usage:"
    echo "       ./getReferenceCarrier.sh <upgrade Carrier>"
    echo "Run in armada-performance/jenkins/carrier/launchdarkly directory"
    echo
    echo "Exiting with error."
    exit 1
fi

upgCarrier=$1

refCarrier=$(cat reference-carrier.json | jq -r '."'"${upgCarrier}"'"')
if [ ${refCarrier} == null ]; then
    echo "No reference carrier found for ${upgCarrier} in reference-carrier.json:"
    cat reference-carrier.json
    echo
    echo "Exiting with error."
    exit 1
fi

# Echo the reference carrier for use by other scripts.
# Do not add any other echo message in this script except when exiting with error.
echo ${refCarrier}
exit 0
