#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Run on carrier master to make config changes to all masters.
# Make sure only testing clusters are running on carrier.
#
# Default KMS cache_timeout is 1.  Setting to 0 will disable caching.

if [ $# -lt 1 ]; then
    echo "Usage: patch_cache_timeout.sh [ 0 | 1 ]"
    exit 1
fi

cache_timeout=$1

clusterIds=$(kubectl get pod -n kubx-masters | grep openvpnserver | sed "s/-/ /g" | awk '{print $2}')

for clusterId in $clusterIds; do
    echo Patching $clusterId
    kubectl patch deploy -n kubx-masters master-$clusterId -p '{"spec":{"template":{"spec":{"containers":[{"name":"kms","env":[{"name":"CACHE_TIMEOUT_IN_HOURS","value":"${cache_timeout}"}]}]}}}}'
    sleep 1
done
