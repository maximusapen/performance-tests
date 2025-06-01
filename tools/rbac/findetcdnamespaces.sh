#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Returns the kubx-etcd-## namespace where the cluster is located
# Usage: ./findetcdnamespaces.sh <cluster id>

CLUSTERID=$1

for i in `seq -w 1 18`; do
	ETCD=$(kubectl -n kubx-etcd-$i get etcdclusters | grep ${CLUSTERID})
	if [[ -n ${ETCD} ]]; then
		echo kubx-etcd-$i
		exit
	fi
done
