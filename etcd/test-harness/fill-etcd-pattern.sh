#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

. etcd-perftest-config

FILL_GB=5
dt=$(date +"%Y-%m-%d-%H-%M")
RESULTS_DIR="defrag/${dt}"

CLUSTERS_PER_GB=74900
PUTS_PER_CLUSTER=52
TOTAL_CLUSTERS=$((FILL_GB*CLUSTERS_PER_GB))
TOTAL_PUTS=$((TOTAL_CLUSTERS*PUTS_PER_CLUSTER))

mkdir -p ${RESULTS_DIR}

echo "Start: ${dt} Generate ${FILL_GB} GB by using pattern to generate keys for ${TOTAL_CLUSTERS} clusters via ${TOTAL_PUTS} puts"
etcdctl --endpoints=${ETCD_ENDPOINTS} endpoint status -w table
etcd-benchmark pattern --endpoints ${ETCD_ENDPOINTS} --clients 20 --conns 20 --clusterid ${TOTAL_CLUSTERS} --masterid 1 --region 1 --workerid 1 --total ${TOTAL_PUTS} --csv-file pattern.csv --file-comment clusters:${TOTAL_CLUSTERS}_total:${TOTAL_PUTS}
etcdctl --endpoints=${ETCD_ENDPOINTS} endpoint status -w table
echo "End: $(date +'%Y-%m-%d-%H-%M')"
