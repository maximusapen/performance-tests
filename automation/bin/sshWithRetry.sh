#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# The ssh command can fail in Jenkins.  This will retry ssh if failed.

maxRetry=10
# sleepTime will increase by 1 min with each retry.
declare -i sleepTime=0

set +e
for i in $(seq 0 ${maxRetry}); do
    ssh "$@"
    if [[ $? == 0 ]]; then
        # ssh command succeeded, continue
        exit 0
    fi
    # ssh command failed.  Sleep with increased interval and retry
    echo "$i: Failed ssh command.  Sleep for ${sleepTime} sec and retry."
    sleep ${sleepTime}
    sleepTime=$((${sleepTime} + 60))
done
set -e

# If we get down to here, ssh has been failing
echo "Failed to run ssh command:"
exit 1
