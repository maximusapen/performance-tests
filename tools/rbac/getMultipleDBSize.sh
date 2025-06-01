#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
export KUBECONFIG=/performance/config/carrier5_stage/admin-kubeconfig
TIME=$(date --utc +%FT%TZ)
echo "Counts at $TIME"
./getClusterEtcdData.sh   fakecruiser-churn-1008 # Baseline
./getClusterEtcdData.sh   fakecruiser-churn-1009 # 100
./getClusterEtcdData.sh   fakecruiser-churn-1010 # 500
./getClusterEtcdData.sh   fakecruiser-churn-1011 # 1000
./getClusterEtcdData.sh   fakecruiser-churn-1012 # 2500
./getClusterEtcdData.sh   fakecruiser-churn-1013 # 5000
