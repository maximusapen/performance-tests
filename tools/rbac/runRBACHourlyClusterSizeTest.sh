#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Determine how much etcd db grows due to loading RBAC users, and record etcd db size every hour for 24 hours.
# Usage: ./runRBACHourlyClusterSizeTest.sh <cluster name> <number of users>
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

echo $KUBECONFIG | grep ${CLUSTERID}
if [[ $? -ne 0 ]]; then
	echo "ERROR: Couldn't set KUBECONFIG to cluster"
	exit 1
fi


SECONDS=0
echo "Adding $USERS users"
./createRBACUserObjects.sh ${USERS} 
DELTA=$SECONDS

DELTA_MIN=$((DELTA/60))
DELTA_SEC=$((DELTA-(DELTA_MIN*60)))

echo "$USERS users added in $DELTA_MIN:$DELTA_SEC (mm:ss) into $CLUSTER_NAME"

export KUBECONFIG=/performance/config/carrier5_stage/admin-kubeconfig
for ((i=0; i<24; i++)); do
	echo "Post adding users: Sizes before compaction/defrag: Hour $i"
	./getClusterEtcdData.sh $CLUSTERID
	sleep 3600
done



#echo "compaction/defrag"
#./compressDefragClusterEtcd.sh $CLUSTERID
#echo "Post adding users: Sizes after compaction/defrag"
#./getClusterEtcdData.sh $CLUSTERID

echo 
