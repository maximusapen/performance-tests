#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Gets the server logs from each member of the cluster

. etcd-perftest-config

dt=$(date +"%Y-%m-%d-%H-%M")
RESULTS_DIR="logs/${dt}"

mkdir -p ${RESULTS_DIR}

echo "Start: ${dt}"
for i in `kubectl -n ${NAMESPACE} get pods --no-headers -l etcd_cluster=${ETCDCLUSTER_NAME} | cut -d" " -f1`; do
    kubectl -n ${NAMESPACE} logs $i -c etcd > ${RESULTS_DIR}/$i.log
done
kubectl -n ${NAMESPACE} get pods -o wide | grep ${ETCDCLUSTER_NAME} > ${RESULTS_DIR}/pods.log
echo "End: $(date +'%Y-%m-%d-%H-%M')"
