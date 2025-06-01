#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

./runRBACClusterSizeTest.sh fakecruiser-churn-1008 0 # Baseline
./runRBACClusterSizeTest.sh fakecruiser-churn-1009 500
./runRBACClusterSizeTest.sh fakecruiser-churn-1010 500
./runRBACClusterSizeTest.sh fakecruiser-churn-1011 1000
./runRBACClusterSizeTest.sh fakecruiser-churn-1012 2500
./runRBACClusterSizeTest.sh fakecruiser-churn-1013 5000
