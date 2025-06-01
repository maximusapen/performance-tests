#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Get the endpoint status of each of the etcd servers via exec to the pod, and also via the endpoint list
# Usage: get-etcd-endpoint-status.sh [true|false]
# If `true` is set then the status of the first etcd server will be output.

. etcd-perftest-config

getSingleStatus=false
if [[ $# -eq 1 ]]; then
    getSingleStatus=$1
fi

dt=$(date +"%Y-%m-%d-%H-%M")

echo "Start: ${dt}"
for i in `kubectl -n ${NAMESPACE} get pods --no-headers -l etcd_cluster=${ETCDCLUSTER_NAME} | cut -d" " -f1`; do
    kubectl -n ${NAMESPACE} exec $i -c etcd -it -- sh -c "ETCDCTL_API=3 etcdctl ${SERVER_ETCDCREDS} --endpoints=127.0.0.1:2379 endpoint status -w table"
    if [[ ${getSingleStatus} == "true" ]]; then
        exit 0
    fi
done
if [[ ${USE_CERTIFICATES} != "true" ]]; then
    etcdctl ${ETCDCREDS} --endpoints=${ETCD_ENDPOINTS} endpoint status -w table
fi
echo "End: $(date +'%Y-%m-%d-%H-%M')"
