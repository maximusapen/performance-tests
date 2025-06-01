#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Determine how much etcd db grows due to loading RBAC users
# Usage: ./runRBACClusterSizeTest.sh <cluster name> <number of users>
# Assumes KUBECONFIG is set to carrier kube

CLUSTER_NAME=$1
USERS=$2

if [[ ${CLUSTER_NAME} == "" || ${CLUSTER_NAME} != "fakecruiser-churn-"* || ${USERS} == "" ]]; then
	echo "ERROR: Parameter error"
	exit 1
fi

CLUSTERID=$(grep "$CLUSTER_NAME " clusters.txt | awk '{print $2}')
echo "CLUSTERID: $CLUSTERID"
if [[ $CLUSTERID == "" ]]; then
	echo "Bad cluster id"
	exit 1
fi

date

echo "Sizes before compaction/defrag"
./getClusterEtcdData.sh $CLUSTERID
echo "compaction/defrag"
./compressDefragClusterEtcd.sh $CLUSTERID
echo "Sizes after compaction/defrag"
./getClusterEtcdData.sh $CLUSTERID
echo "Load kubeconfig"
. setPerfKubeconfig.sh ${CLUSTERID}
SECONDS=0
echo "Adding $USERS users"
./createRBACUserObjects.sh ${USERS} 
DELTA=$SECONDS

DELTA_MIN=$((DELTA/60))
DELTA_SEC=$((DELTA-(DELTA_MIN*60)))

echo "$USERS users added in $DELTA_MIN:$DELTA_SEC (mm:ss) into $CLUSTER_NAME"

export KUBECONFIG=/performance/config/carrier5_stage/admin-kubeconfig
echo "Post adding users: Sizes before compaction/defrag"
./getClusterEtcdData.sh $CLUSTERID
echo "compaction/defrag"
./compressDefragClusterEtcd.sh $CLUSTERID
echo "Post adding users: Sizes after compaction/defrag"
./getClusterEtcdData.sh $CLUSTERID

echo 
