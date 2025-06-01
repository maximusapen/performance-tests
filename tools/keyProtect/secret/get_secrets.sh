#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019, 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

if [ $# -lt 2 ]; then
    echo "Usage: get_secrets.sh <start kpcluster no> <end kpcluster no>"
    exit 1
fi

# clusterStart and clusterEnd set by get_secrets_6.sh.  Not by test_conf.sh
clusterStart=$1
clusterEnd=$2

date
for i in $(seq ${clusterStart} ${clusterEnd}); do
    cluster=kpcluster${i}
    export KUBECONFIG=$HOME/.bluemix/plugins/container-service/clusters/${cluster}/kube-config-dal09-${cluster}.yml
    echo
    echo Cluster: ${cluster}
    echo Start:
    date
    # Multiple attempts to get secret.  First few attempts may timeout if many DEKs
    for j in $(seq 1 5); do
        SECONDS=0
        nSecrets=$(kubectl get secret --all-namespaces | grep -v NAME | wc -l)
        echo "$j-${cluster}:  Time taken to get ${nSecrets} secrets: ${SECONDS} sec"
        if [[ ${nSecrets} == "0" ]]; then
            # Getting Error.  Sleep for a bit to avoid hitting the api too soon
            sleep 5
        fi
    done
    echo End:
    date
done
date
