#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Loads the specified number of RBAC users concurrently into a set of clusters.
# Monitors `kubectl get svc` times for each cluster before, during and after the test.
# USAGE: ./createRBACUserObjectsConcurrent.sh <start cruiser postfix number> <end cruiser postfix number> <number of users>
# It was a manual operation to parse get svc times from logs and create statistics for before, during and after the test run.

CLUSTER_PREFIX="fakecruiser-churn-"
START_USER=$1
END_USER=$2
USERS=$3

if [[ ${START_USER} == "" || ${END_USER} == "" || ${START_USER} -gt ${END_USER} ]]; then
	echo "ERROR: Parameter error"
	exit 1
fi

date

for (( i=${START_USER}; i<=${END_USER}; i++ )); do
        CLUSTER=${CLUSTER_PREFIX}${i}
	./checkServiceTime.sh $CLUSTER 10 >> monitor.$CLUSTER.log &
done

sleep 60

for (( i=${START_USER}; i<=${END_USER}; i++ )); do
        CLUSTER=${CLUSTER_PREFIX}${i}
        echo "Load kubeconfig for ${CLUSTER}"
        . setPerfKubeconfig.sh ${CLUSTER}
	./createRBACUserObjects.sh ${USERS} &
done

echo "Waiting for users to be created and monitoring to end"
wait

date

echo "------------------------------------"
