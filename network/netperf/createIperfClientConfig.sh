#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright Maximus Apen, 2025 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Creates netperf-pod#.yml files based on netperf-pod1.yml and deploys them if requested

namespace=iperf
deploy=false
recreate=false
lb_postfix=""

function deployIperfClient {
    podExists=false
    if [[ $deploy == "true" ]]; then
        echo "Deploying netperf-pod$1${lb_postfix}.yml"
        DELAY=0
        kubectl -n $namespace get pod netperf-pod$1${lb_postfix} > /dev/null 2>&1
        if [[ $? -eq 0 ]]; then
            if [[ $recreate == "true" ]]; then
                echo "Deleting existing pod"
                kubectl -n $namespace delete pod netperf-pod$1${lb_postfix} > /dev/null 2>&1
                DELAY=60
                echo "Sleeping for $DELAY seconds to insure deletes completed"
                sleep $DELAY
            else
                podExists=true
            fi
        fi

        if [[ $podExists == "false" ]]; then
            echo "Deploying netperf-pod$1${lb_postfix}.yml"
            kubectl -n $namespace create -f netperf-pod$1${lb_postfix}.yml
        fi
    fi
}

if [[ $# -lt 0 ]]; then
    echo "USAGE: createIperfClientConfig.sh <number of netperf-pod#.yml files to create from netperf-pod1.yml template> <'true' if pod should be deployed> <'true' to delete and recreate existing iperf clients> <'true' to create loadbalancer pods>"
elif [[ $# -lt 1 ]]; then
    echo "ERROR: Count must be > 1"
fi

if [[ $# -ge 2 && $2 == "true" ]]; then
    deploy=true
fi

if [[ $# -ge 3 && $3 == "true" ]]; then
    recreate=true
fi

if [[ $# -ge 4 && $4 == "true" ]]; then
    lb_postfix="-lb"
fi

if [[ -n ${lb_postfix} ]]; then
    sed -e "s/netperf-pod1/netperf-pod01${lb_postfix}/g" netperf-pod1.yml > netperf-pod01${lb_postfix}.yml
fi
deployIperfClient "01"

for ((i=2; i<=$1; i++)); do
    pod_id=$(seq -f '%02g' $i $i)
    echo "Creating netperf-pod${pod_id}${lb_postfix}.yml"
    sed -e "s/netperf-pod1/netperf-pod${pod_id}${lb_postfix}/g" netperf-pod1.yml > netperf-pod${pod_id}${lb_postfix}.yml
    deployIperfClient ${pod_id}
done

kubectl -n ${namespace} get pods -o wide
