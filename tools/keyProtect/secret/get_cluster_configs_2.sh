#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019, 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Run after logging into IKS test carrier and after all clusters are in normal state

clusterStart=1
clusterEnd=950

date
for i in $(seq ${clusterStart} ${clusterEnd}); do
	cluster=kpcluster${i}
	echo Cluster: ${cluster}
	ibmcloud ks cluster config --cluster ${cluster} -s
done
date
