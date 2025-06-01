#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Run after logging into IKS test carrier and after generate_secrets_4.sh
#
# This script is called by add_secrets_*.sh to add secrets in parallel.
# Or you can all this script by changing clusterStart=1 and clusterEnd=80 to run in sequence
# which takes more time.
if [ $# -lt 4 ]; then
    echo "Usage: add_secrets.sh <start kpcluster no> <end kpcluster no> <start secret no> <end secret no>"
    exit 1
fi

# clusterStart and clusterEnd set by get_secrets_6.sh.  Not by test_conf.sh
clusterStart=$1
clusterEnd=$2
secretStart=$3
secretEnd=$4

date
for i in $(seq ${clusterStart} ${clusterEnd}); do
    cluster=kpcluster${i}
    export KUBECONFIG=$HOME/.bluemix/plugins/container-service/clusters/${cluster}/kube-config-dal09-${cluster}.yml
    echo Cluster: ${cluster}
    echo Start:
    date
    for j in $(seq ${secretStart} ${secretEnd}); do
        kubectl apply -f yaml/perf-${i}-secret-${j}.yaml
    done
    echo End:
    date
done
date
