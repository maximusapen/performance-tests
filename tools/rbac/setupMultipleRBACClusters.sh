#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Setup clusters with different number of users

./setupRBACCluster.sh fakecruiser-churn-1021 100
./setupRBACCluster.sh fakecruiser-churn-1022 500
./setupRBACCluster.sh fakecruiser-churn-1023 1000
./setupRBACCluster.sh fakecruiser-churn-1024 2500
./setupRBACCluster.sh fakecruiser-churn-1025 5000
