#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Get the master status of all test clusters
# Run after logging into IKS test carrier.

if [ $# -lt 2 ]; then
    echo "Usage: get_master_status.sh <start kpcluster no> <end kpcluster no>"
    exit 1
fi

# clusterStart and clusterEnd set by get_secrets_6.sh.  Not by test_conf.sh
clusterStart=$1
clusterEnd=$2

date
for i in $(seq ${clusterStart} ${clusterEnd}); do
    cluster=kpcluster${i}
    masterStatus=$(ibmcloud ks cluster get ${cluster} | grep "Master Status:")
    echo "${cluster} - ${masterStatus}"
done
date
