#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright Maximus Apen, 2025 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Run after logging into IKS test carrier.
#
# Check Master Status for specified KP cluster #

startCluster=1
endCluster=80

date
for i in $(seq ${startCluster} ${endCluster}); do
    cluster=kpcluster${i}
    echo cluster: ${cluster}
    ibmcloud ks cluster get --cluster ${cluster} | grep "Master Status"
done
date
