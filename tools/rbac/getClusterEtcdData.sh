#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Gets the database size of each etcd server in the etcd clusters
# Usage: ./getClusterEtcdData.sh <cluster name (must be "fakecruiser-churn-*")|cluster id>

if [[ $1 == "fakecruiser-churn-"* ]]; then
	CLUSTERID=$(grep "$1 " clusters.txt | awk '{print $2}')
	echo "CLUSTERID: $CLUSTERID"
	if [[ $CLUSTERID == "" ]]; then
	        echo "Bad cluster id"
       		exit 1
	fi
	echo "Get etcd db sizes for cluster $CLUSTERID - $1"
else
	CLUSTERID=$1
	echo "Get etcd db sizes for cluster $CLUSTERID"
fi


ETCD_NS=$(./findetcdnamespaces.sh ${CLUSTERID})
if [[ -z ${ETCD_NS} ]]; then
	echo "Didn't find etcd namespace"
	exit 1
fi
for POD in $(kubectl -n ${ETCD_NS} get pods -l etcd_cluster=etcd-${CLUSTERID} --no-headers -o custom-columns=NAME:.metadata.name); do
	echo ${ETCD_NS} ${POD}
	#kubectl -n ${ETCD_NS} exec ${POD} -c etcd -it -- etcdctl endpoint status --cacert=/etc/etcdtls/member/server-tls/server-ca.crt --cert=/etc/etcdtls/member/server-tls/server.crt --key=/etc/etcdtls/member/server-tls/server.key -w table
	kubectl -n ${ETCD_NS} exec ${POD} -c etcd -- etcdctl endpoint status --cacert=/etc/etcdtls/member/server-tls/server-ca.crt --cert=/etc/etcdtls/member/server-tls/server.crt --key=/etc/etcdtls/member/server-tls/server.key -w table
done
