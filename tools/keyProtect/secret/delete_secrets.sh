#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright Maximus Apen, 2025 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Run after logging into IKS test carrier.
#
# This script is called by del_all_secrets.sh to delete secrets in parallel.
# Or you can all this script by changing start=1 and end=80 to run in sequence
# which takes more time.
#
# Check and modify "secretEnd" if total number of secrets are different.

if [ $# -lt 4 ]; then
	echo "Usage: delete_secrets.sh <start kpcluster no> <end kpcluster no> <start secret no> <end secret no>"
	exit 1
fi

# clusterStart and clusterEnd set by delete_secrets_6.sh.  Not by test_conf.sh
clusterStart=$1
clusterEnd=$2
secretStart=$3
secretEnd=$4

date
for i in $(seq ${clusterStart} ${clusterEnd}); do
	cluster=kpcluster${i}
	export KUBECONFIG=$HOME/.bluemix/plugins/container-service/clusters/${cluster}/kube-config-dal09-${cluster}.yml
	echo Cluster: ${cluster}
	echo Start:
	date
	for j in $(seq ${secretStart} ${secretEnd}); do
		kubectl delete -f yaml/perf-${i}-secret-${j}.yaml
	done
	echo End:
	date
done
date
