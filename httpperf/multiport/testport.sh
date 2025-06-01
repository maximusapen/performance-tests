#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Simple manual test provided to measure how long it takes for all 1000 ports to be available
# after multiport service is created.

waitForConnection() {
    host=$1
    port=$2
    declare -i counter=1
    declare -i maxCounter=24
    SECONDS=0
    printf "%s - Checking ${host}:${port} connection.\n" "$(date +%T)"
    while true; do
        set +e
        resp=$(nc -zv -w10 "${host}" "${port}" 2>&1)
        set -e
        if [[ ${resp} == *"succeeded"* ]]; then
            # During testing, it has been found that connection can succeed and then drop off.
            # Repeating this check for ${maxCounter} times before considered it stable.
            if [[ counter -eq 1 ]]; then
                duration=${SECONDS}
            fi
            if [[ counter -eq ${maxCounter} ]]; then
                duration=${SECONDS}
                printf "%s -   Verified connection to ${host}:${port} in ${duration} sec.\n" "$(date +%T)"
                break
            fi
            ((counter++))
        else
            # Reset counter to 1 once connection failed.
            # Testing has shown that the connection can take over an hour to succeed.
            # Fail test after 2 hours.
            if [[ ${SECONDS} -gt 7200 ]]; then
                printf "%s -   Waited connection to ${host}:${port} for over 2 hours.  Failing test.\n" "$(date +%T)"
                exit 1
            fi
            counter=1
            printf "%s -   Waiting for %s %s to become available.\n" "$(date +%T)" "${host}" "${port}"
            sleep 60
        fi
    done
}

host=$1
declare -i port=30091
nPort=1000

SECONDS=0
for i in $(seq 1 ${nPort}); do
    waitForConnection ${host} ${port}
    port=${port}+1
done
duration=${SECONDS}
printf "%s - Time taken to check all ports in multi-port service is $(($duration / 60)) minutes and $(($duration % 60)) seconds.\n" "$(date +%T)"
