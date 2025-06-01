#!/bin/bash -e


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
