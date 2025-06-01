#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Runs etcd compaction on each etcd server and then defrag on the cluster
# Usage: ./compressDefragClusterEtcd.sh <cluster name>
# Assumes KUBECONFIG set to carrier kube.

CLUSTERID=$1
ETCD_NS=$(./findetcdnamespaces.sh ${CLUSTERID})
if [[ -z ${ETCD_NS} ]]; then
	echo "Didn't find etcd namespace"
	exit 1
fi
FIRSTPOD=true
for POD in $(kubectl -n ${ETCD_NS} get pods -l etcd_cluster=etcd-${CLUSTERID} --no-headers -o custom-columns=NAME:.metadata.name); do
	echo ${ETCD_NS} ${POD}
	if [[ ${FIRSTPOD} == "true" ]]; then
		FIRSTPOD=false
		#revision=$(${ETCD} --endpoints="$ETCD_ENDPOINTS" $CREDS --command-timeout=100s endpoint status --write-out="json" | egrep -o '"revision":[0-9]*' | egrep -o '[0-9]*' | sort | head -1)
		revision=$(kubectl -n ${ETCD_NS} exec ${POD} -c etcd -- etcdctl --cacert=/etc/etcdtls/member/server-tls/server-ca.crt --cert=/etc/etcdtls/member/server-tls/server.crt --key=/etc/etcdtls/member/server-tls/server.key --command-timeout=100s endpoint status --write-out="json" | egrep -o '"revision":[0-9]*' | egrep -o '[0-9]*' | sort | head -1)
		echo "revision=$revision"
		kubectl -n ${ETCD_NS} exec ${POD} -c etcd -- etcdctl --cacert=/etc/etcdtls/member/server-tls/server-ca.crt --cert=/etc/etcdtls/member/server-tls/server.crt --key=/etc/etcdtls/member/server-tls/server.key --command-timeout=100s compact ${revision}
	fi
	kubectl -n ${ETCD_NS} exec ${POD} -c etcd -- etcdctl --cacert=/etc/etcdtls/member/server-tls/server-ca.crt --cert=/etc/etcdtls/member/server-tls/server.crt --key=/etc/etcdtls/member/server-tls/server.key  --command-timeout=100s defrag
done
