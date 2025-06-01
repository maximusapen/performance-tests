#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Run on carrier master to make config changes to all masters.
# Make sure only testing clusters are running on carrier.
#
# Update <cache_size> in cmpatch.yaml before running this script.
#
# Default kms <cache_size> is 100 and is hard-coded in
# https://github.ibm.com/alchemy-containers/armada-ansible/blob/master/kubX/roles/master-service/templates/kms-kubeconfig.j2
# Setting to 0 will turn off cache.

kubectl get cm -n kubx-masters master-$clusterid-config -o yaml

clusterIds=$(kubectl get pod -n kubx-masters | grep openvpnserver | sed "s/-/ /g" | awk '{print $2}')

for clusterId in $clusterIds; do
    echo Patching cm for $clusterId
    kubectl patch cm -n kubx-masters master-$clusterId-config --patch "$(cat cmpatch.yaml)"
done
