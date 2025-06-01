#!/bin/bash -e

# Git clone failed frequently fails for Gits with frequent updates.  This will retry git clone if failed.

maxRetry=10
sleepTime=300

if [[ $# -lt 2 ]]; then
    echo
    echo "Usage: ./gitCloneWithRetries.sh <git> <cloned_directory> [max_number_of_retries] [sleepTime_between_retries]"
    echo
    echo "Example: To git clone cfs-inventroy with default 10 retries and default 300s sleep between retries "
    echo "    ./gitCloneWithRetries.sh git@github.ibm.com:alchemy-conductors/cfs-inventory.git cfs-inventory"
    echo
    echo "By default, cloned_directory is in WORKSPACE.  If you need to clone to a different directory"
    echo "such as src/github.ibm.com/alchemy-containers/armada-performance, you need to cd to"
    echo "WORKSPACE/src/github.ibm.com/alchemy-containers before calling this script."
    echo

    exit 1
fi

gitToClone=$1
gitDir=$2

if [[ ! -z $3 ]]; then
    maxRetry=$3
    if [[ ! -z $4 ]]; then
        sleepTime=$4
    fi
fi

echo "Git clone ${gitToClone} to ${gitDir}.  maxRetry: ${maxRetry}.  sleepTime: ${sleepTime}"
set +e
for i in $(seq 0 ${maxRetry}); do
    git clone --depth 1 --single-branch --branch=master ${gitToClone}
    GIT_CLONE_RESULT=$?
    echo "GIT_CLONE_RESULT: ${GIT_CLONE_RESULT}"
    if [[ ${GIT_CLONE_RESULT} == 0 ]]; then
        # Git clone succeeded, continue
        break
    fi
    # Git clone failed.  Sleep and retry
    SLEEP_TIME=300
    echo "$i: Failed to clone cfs-inventory.  Sleep for ${SLEEP_TIME} sec and retry."
    sleep ${SLEEP_TIME}
done
set -e

if [[ ! -d ${gitDir} ]]; then
    echo "Failed to clone ${gitToClone}"
    exit 1
fi
