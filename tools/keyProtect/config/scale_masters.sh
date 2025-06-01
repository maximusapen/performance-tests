#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Run on carrier master to scale all masters.
# If testing, make sure only testing clusters are running on carrier.
#
# Default replicas is 3 for clusters in multi-zone carrier.  Setting to 0 will delete all master pods.
# Usually used in emergency mode to set replicas to 0 and then reset to 3 after emergency.
# Save the output of `patch_cache_timeout.sh 0` for scaling up later.

if [[ $# -lt 1 ]]; then
    echo "Usage: scale_masters.sh [ 0 | 3 ]"
    exit 1
fi

source ../test_conf.sh

replicas=$1
declare -i totalScaledPods=${numCluster}*${replicas}

export KUBECONFIG=/performance/config/${carrier}_stage/admin-kubeconfig

# If only KP clusters exists on carrier, you can use openvpnserver pods to get the KP cluster list
#clusterIds=$(kubectl get pod -n kubx-masters | grep openvpnserver | sed "s/-/ /g" | awk '{print $2}')

# Search for 5/5 master pods if non-KP clusters also exists on carrier
#clusterIds=$(kubectl get pod -n kubx-masters | grep "5/5" | sed "s/-/ /g" | awk '{print $2}')

# Scale clusters up that have been scaled down to 0.  Save the output as scale0.log when scaling clusters down to 0
#clusterIds=$(cat scale0.log | grep Scale | awk '{print $2}')
clusterIds=$(ibmcloud ks clusters | grep "kpcluster" | awk '{print $2}')

lastClusterId=""
date
for clusterId in $clusterIds; do
    if [[ $clusterId != $lastClusterId ]]; then
        echo Scale $clusterId
        kubectl scale deployment -n kubx-masters master-$clusterId --replicas ${replicas}
        lastClusterId=$clusterId
        sleep 1
    fi
done
date

date
echo "Check all clusters are scaled"
while true; do
    nPods=$(kubectl get pod -n kubx-masters | grep "/5" | wc -l)
    echo "Number of master pods: ${nPods}"
    if [[ ${nPods} == ${totalScaledPods} ]]; then
        echo "All pods now scaled to ${replicas}"
        break
    fi
done
echo Finished
date
