#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Preloads fakecruiser-churn-# kubeconfigs
# Usage: loadKubeConfigs.sh <start cluster number> <end cluster number>

START_USER=$1
END_USER=$2
for (( i=${START_USER}; i<=${END_USER}; i++ )); do
	CLUSTER=fakecruiser-churn-$i
	echo "Load kubeconfig for ${CLUSTER}"
	. setPerfKubeconfig.sh ${CLUSTER}
done
