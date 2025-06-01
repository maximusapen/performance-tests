#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Defrag, and optionally compact, etcd cluster: defrag-etcd.sh <compact>

. etcd-perftest-config

dt=$(date +"%Y-%m-%d-%H-%M")
RESULTS_DIR="results-defrag/${dt}"

mkdir -p ${RESULTS_DIR}

#echo "Start: ${dt}"
./get-etcd-endpoint-status.sh true
if [[ $1 == "compact" ]]; then
    if [[ ${USE_CERTIFICATES} == "true" ]]; then
        pod=$(kubectl -n ${NAMESPACE} get pods --no-headers -l etcd_cluster=${ETCDCLUSTER_NAME} | head -1 | cut -d" " -f1)
        if [[ -z ${pod} ]]; then
            echo "ERROR: Couldn't find test etcd pod"
            exit 1
        fi
        endpointStatus=$(kubectl -n ${NAMESPACE} exec ${pod} -c etcd -- sh -c "ETCDCTL_API=3 etcdctl ${SERVER_ETCDCREDS} --endpoints=127.0.0.1:2379 endpoint status --write-out=\"json\"")
    else
        endpointStatus=$(etcdctl $ETCDCREDS --endpoints $ETCD_ENDPOINTS endpoint status --write-out="json")
    fi
    rev=$(echo ${endpointStatus} | egrep -o '"revision":[0-9]*' | egrep -o '[0-9]+' | head -1)
    echo "Revision: ${rev}"
    if [[ ${USE_CERTIFICATES} == "true" ]]; then
        echo "Compacting via ${pod}"
        result=$(kubectl -n ${NAMESPACE} exec ${pod} -c etcd -- sh -c "ETCDCTL_API=3 etcdctl ${SERVER_ETCDCREDS} --endpoints=127.0.0.1:2379 --dial-timeout ${DEFRAG_TIMEOUT} --command-timeout ${DEFRAG_TIMEOUT} compact $rev" 2>&1)
    else
        result=$(etcdctl $ETCDCREDS --endpoints $ETCD_ENDPOINTS --dial-timeout ${DEFRAG_TIMEOUT} --command-timeout ${DEFRAG_TIMEOUT} compact $rev 2>&1)
    fi
    if [[ $? -ne 0 ]]; then
        if [[ ${result} =~ "too many requests" ]]; then
            # In cases where "too many requests" error occurs, etcd may return an error
            echo "Sleeping to clear: ${result}"
            sleep 30
            if [[ ${USE_CERTIFICATES} == "true" ]]; then
                kubectl -n ${NAMESPACE} exec ${pod} -c etcd -- sh -c "ETCDCTL_API=3 etcdctl ${SERVER_ETCDCREDS} --endpoints=127.0.0.1:2379 --dial-timeout ${DEFRAG_TIMEOUT} --command-timeout ${DEFRAG_TIMEOUT} compact $rev"
            else
                etcdctl $ETCDCREDS --endpoints $ETCD_ENDPOINTS --dial-timeout ${DEFRAG_TIMEOUT} --command-timeout ${DEFRAG_TIMEOUT} compact $rev
            fi
            sleep ${COMPACT_WAIT}
        else
            echo "${result}"
        fi
    else
        sleep ${COMPACT_WAIT}
    fi
    # Need time between compact and defrag requests since compaction can take some time.
fi
for i in `kubectl -n ${NAMESPACE} get pods --no-headers -l etcd_cluster=${ETCDCLUSTER_NAME} | cut -d" " -f1`; do
    echo "Defraging via ${i}"
    if [[ ${USE_CERTIFICATES} == "true" ]]; then
        kubectl -n ${NAMESPACE} exec $i -c etcd -- sh -c "ETCDCTL_API=3 etcdctl ${SERVER_ETCDCREDS} --endpoints=127.0.0.1:2379 --dial-timeout ${DEFRAG_TIMEOUT} --command-timeout ${DEFRAG_TIMEOUT} defrag"
    else
        kubectl -n ${NAMESPACE} exec $i -c etcd -- sh -c "ETCDCTL_API=3 etcdctl --endpoints=127.0.0.1:2379 --dial-timeout ${DEFRAG_TIMEOUT} --command-timeout ${DEFRAG_TIMEOUT} defrag"
    fi
    kubectl -n ${NAMESPACE} logs $i -c etcd > ${RESULTS_DIR}/$i.log
done
echo "Disarming alarms"
if [[ ${USE_CERTIFICATES} == "true" ]]; then
    kubectl -n ${NAMESPACE} exec ${pod} -c etcd -- sh -c "ETCDCTL_API=3 etcdctl ${SERVER_ETCDCREDS} --endpoints=127.0.0.1:2379 alarm disarm"
else
    etcdctl $ETCDCREDS --endpoints=${ETCD_ENDPOINTS} alarm disarm
fi
./get-etcd-endpoint-status.sh
#echo "End: $(date +'%Y-%m-%d-%H-%M')"
