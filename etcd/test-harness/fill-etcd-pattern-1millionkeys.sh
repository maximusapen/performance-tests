#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

. etcd-perftest-config


PATTERN_FILE="--patternFile etcd.onemillionkeys.fast.pattern.txt"
RATE_LIMIT=" --put-rate 100"
RATE_LIMIT=
dt=$(date +"%Y-%m-%d-%H-%M")
RESULTS_DIR="defrag/${dt}"

CLIENTS=300
CONNECTIONS=${CLIENTS}
TOTAL_PUTS=1000000

mkdir -p ${RESULTS_DIR}

echo "Start: Generate ${TOTAL_PUTS} keys with ${CLIENTS} clients and ${CONNECTIONS} connections. etcd v$(kubectl -n armada get etcdclusters ${ETCDCLUSTER_NAME} -o jsonpath='{.spec.version}'), ${RATE_LIMIT} ${PATTERN_FILE}"
etcdctl --endpoints=${ETCD_ENDPOINTS} endpoint status -w table
./etcd-benchmark pattern --endpoints ${ETCD_ENDPOINTS} --clients ${CLIENTS} --conns ${CONNECTIONS} --clusterid 1 --masterid ${TOTAL_PUTS} --region 1 --workerid 1 --total ${TOTAL_PUTS} --csv-file onemillionkeys.csv --file-comment puts:${TOTAL_PUTS}-clients:${CLIENTS}-conn:${CONNECTIONS}  ${RATE_LIMIT} ${PATTERN_FILE}
etcdctl del  $ETCDCREDS --endpoints $ETCD_ENDPOINTS --dial-timeout ${DEFRAG_TIMEOUT} --command-timeout ${DEFRAG_TIMEOUT} --prefix /onemillionkeys
./defrag-etcd.sh compact
echo "End: $(date +'%Y-%m-%d-%H-%M')"
