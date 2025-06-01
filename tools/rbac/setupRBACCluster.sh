#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# This sript adds the specified number of RBAC users to the specified cluster
# Usage: ./setupRBACCluster.sh <cluster name> <number of users>

CLUSTER_NAME=$1
USERS=$2

if [[ ${CLUSTER_NAME} == "" || ${CLUSTER_NAME} != "fakecruiser-churn-"* || ${USERS} == "" ]]; then
	echo "ERROR: Parameter error"
	exit 1
fi

if [[ ! -f clusters.txt ]]; then
	echo "ERROR: clusters.txt must exist for this script to succeed"
	echo "Run: ibmcloud ks clusters > clusters.txt"
	exit 1
fi

CLUSTERID=$(grep "$CLUSTER_NAME " clusters.txt | awk '{print $2}')
echo "CLUSTERID: $CLUSTERID"
if [[ $CLUSTERID == "" ]]; then
	echo "Bad cluster id. This can occur because clusters.txt is out of date"
	exit 1
fi

date

echo "Load kubeconfig"
. setPerfKubeconfig.sh ${CLUSTERID}
SECONDS=0
echo "Adding $USERS users"
./createRBACUserObjects.sh ${USERS} 
DELTA=$SECONDS

DELTA_MIN=$((DELTA/60))
DELTA_SEC=$((DELTA-(DELTA_MIN*60)))

echo "$USERS users added in $DELTA_MIN:$DELTA_SEC (mm:ss) into $CLUSTER_NAME"

