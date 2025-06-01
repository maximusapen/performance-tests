#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Run after logging into IKS test carrier
#
# Check and change kubeVersion for your test before running this script.
#

clusterStart=1
clusterEnd=950

date
for i in $(seq ${clusterStart} ${clusterEnd}); do
    cluster=kpcluster${i}
    echo cluster: ${cluster}
    ibmcloud ks cluster rm --cluster ${cluster} --force-delete-storage
done
date
