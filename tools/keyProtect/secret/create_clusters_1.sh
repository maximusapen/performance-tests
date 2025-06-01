#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Run after logging into IKS test carrier or set KUBECONFIG to carrier
#
# Check and change kubeVersion for your test before running this script.
#
# Proceed to enable_kp_2.sh when all clusters are running in normal state.

clusterStart=1
clusterEnd=950

# If creating clusters for a specified Kube version and not the default,
# set the kubeVersion below and add "--kube-version "${kubeVersion}" to
# "ibmcloud ks cluster create...." command.

#kubeVersion="1.15"

date
for i in $(seq ${clusterStart} ${clusterEnd}); do
    cluster=kpcluster${i}
    echo cluster: ${cluster}
    # Create 0-worker cluster with "--workers -1", not "--workers 0"
    # Use "--workers 1" if wants to include workers
    ibmcloud ks cluster create --name ${cluster} --no-subnet --location dal09 --machine-type u3c.2x4 --private-vlan 2263901 --public-vlan 2263903 --workers -1
done
date
