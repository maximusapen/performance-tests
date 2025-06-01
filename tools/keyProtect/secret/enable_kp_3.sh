#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019, 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Run after logging into IKS test carrier.
#
# Update clusterStart, clusterEnd, kpInstance and crk before running this script.
# See README.md for creating kpInstance and crk.
#
# Checks after running this script:
#
# Normally takes about 20 minutes for each cluster to complete key protect configuration.
# Configuration is completed when Master status is Ready from "ibmcloud ks cluster get kpcluster1"
# On carrier, all master pods of test clusters are running with 5 containers instead of 4 with additional kms container.

# The 950-clusters load testing is sharing 5 kpInstance/crk.
# Modify clusterStart and clusterEnd to target different cluster set.
clusterStart=1
clusterEnd=200

# Do not add quotes to enclose these parameters which can cause key-protect-enable to fail
kpInstance=< key protect instance GUID >
crk=< customer root key >

date
for i in $(seq ${clusterStart} ${clusterEnd}); do
    cluster=kpcluster${i}
    echo cluster: ${cluster}
    ibmcloud ks kms enable -c ${cluster} --instance-id ${kpInstance} --crk ${crk}
done
date
