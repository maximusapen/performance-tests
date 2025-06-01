#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

. etcd-perftest-config


PATTERN_FILE="--patternFile etcd.77-8000.fast.pattern.txt"
RATE_LIMIT=" --put-rate 100"
RATE_LIMIT=
FILL_GB=5
dt=$(date +"%Y-%m-%d-%H-%M")
RESULTS_DIR="defrag/${dt}"

CLUSTERS_PER_GB=120192 # AugT4
CLUSTERS_PER_GB=120310
PUTS_PER_CLUSTER=1
TOTAL_CLUSTERS=$((FILL_GB*CLUSTERS_PER_GB))
TOTAL_PUTS=$((TOTAL_CLUSTERS*PUTS_PER_CLUSTER))

mkdir -p ${RESULTS_DIR}

echo "Start: ${dt} Generate ${FILL_GB} GB by using pattern to generate keys for ${TOTAL_CLUSTERS} clusters via ${TOTAL_PUTS} puts with etcd v$(kubectl -n armada get etcdclusters ${ETCDCLUSTER_NAME} -o jsonpath='{.spec.version}'), ${RATE_LIMIT} ${PATTERN_FILE}"
etcdctl --endpoints=${ETCD_ENDPOINTS} endpoint status -w table
./etcd-benchmark pattern --endpoints ${ETCD_ENDPOINTS} --clients 20 --conns 20 --clusterid ${TOTAL_CLUSTERS} --masterid 1 --region 1 --workerid 1 --total ${TOTAL_PUTS} --csv-file pattern.csv --file-comment clusters:${TOTAL_CLUSTERS}_total:${TOTAL_PUTS} ${RATE_LIMIT} ${PATTERN_FILE}
etcdctl --endpoints=${ETCD_ENDPOINTS} endpoint status -w table
echo "End: $(date +'%Y-%m-%d-%H-%M')"
