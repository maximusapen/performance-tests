#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# Activate or deactive perf clients in
# https://github.ibm.com/alchemy-containers/armada-performance-data/tree/master/automation/client.json.

# Calling script or Jenkins job responsible for merge/push to update armada-performance-data.
# See Jenkins job https://alchemy-testing-jenkins.swg-devops.com/view/Armada-performance/job/Armada-Performance/job/Automation/job/Activate-Deactivate-Test-Clients/
# on how to clone armada-performance-data, run this script, and then run automation/bin/mergeArmadaPerformanceData.sh
# to update armada-performance-data GIT.

if [ $# -lt 2 ]; then
    echo "Usage:"
    echo "    ./activatePerfClient.sh < perf client > < Activate | Deactivate >"
    echo "Example:"
    echo "    To disable all testing on stage-dal09-perf4-client-01:"
    echo "        ./activatePerfClient stage-dal09-perf4-client-01 Deactivate"
    echo "    To enable all testing on stage-dal09-perf4-client-01:"
    echo "        ./activatePerfClient stage-dal09-perf4-client-01 Activate"
    exit 1
fi

perfClient=$1
action=$2
slackFile=$4

if [[ ${action} == 'Activate' ]]; then
    activate=true
elif [[ ${action} == 'Deactivate' ]]; then
    activate=false
else
    echo "Please specifiy Activate or Deactivate for client ${perfClient}"
    echo "Usage:"
    echo "    ./activatePerfClient.sh < perf client > < Activate | Deactivate >"
    exit 1
fi

clientFile=${WORKSPACE}/armada-performance-data/automation/client.json

activeState=$(jq -r '."'"${perfClient}"'".active' ${clientFile})
echo "Perf client ${perfClient} active state is currently ${activeState}."

if [[ ${activeState} == ${activate} ]]; then
    echo "Perf Client ${perfClient} active state is already ${activate}.  No action required."
    exit 0
fi

# Now update active state
newClientState=$(cat ${clientFile} | jq '."'"${perfClient}"'".active = '${activate}'')
echo ${newClientState} | jq . >${clientFile}

# Check file is updated
newActiveState=$(jq -r '."'"${perfClient}"'".active' ${clientFile})

if [[ ${newActiveState} == ${activate} ]]; then
    echo "Perf client ${perfClient} active state is updated to ${newActiveState}."
else
    echo "Failed to update perf client ${perfClient} active state to ${activate}.  Active state is still ${newActiveState}."
    exit 1
fi

exit 0
