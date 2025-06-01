#!/bin/bash -ex
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

upgCarrier=$1

if [[ $2 == *carrier* ]]; then
    refCarrier=$2
    upgMode=$3 # --upgrade | --query | --upgrade-all | --query-all
else
    upgMode=$2
fi

echo "Upgrade mode is ${upgMode}"

if [ -z "${upgCarrier}" ]; then
    echo "No upgrade carrier."
    echo "Usage: For upgrade mode:"
    echo "       ./upgradeCarrier.sh <update Carrier> [<reference Carrier> | --upgrade | --query | --upgrade-all | --query-all ]"
    echo "       e.g. ./upgradeCarrier.sh stage-dal09-carrier4"
    echo "       e.g. ./upgradeCarrier.sh stage-dal09-carrier4 stage-dal10-carrier0"
    echo "       e.g. ./upgradeCarrier.sh stage-dal09-carrier401 --update-all"
    echo "Run in armada-performance/jenkins/carrier/launchdarkly directory"

    echo "Exiting with error."
    exit 1
fi

if [[ -z "${refCarrier}" ]]; then
    echo
    echo "No reference Carrier."
    echo "Checking default reference Carrier in refCarrier-key.json."
    refCarrier=$(./getReferenceCarrier.sh ${upgCarrier})
    echo "Reference carrier for ${upgCarrier}: ${refCarrier}"
    # Put refCarrier in /tmp for Jenkins job description
    echo "REFERENCE_CARRIER=${refCarrier}" >>/tmp/JJEnv.txt
    cat /tmp/JJEnv.txt
fi

# If not running this script in Jenkins, set LD_API_KEY to
# STAGE_GLOBAL_ARMPERF_LD_APIKEY from Vault
#export LD_API_KEY=${STAGE_GLOBAL_ARMPERF_LD_APIKEY}

./getCarrierLDFlags.sh $upgCarrier $refCarrier
./deployChanges.sh $upgCarrier $refCarrier ${upgMode}
