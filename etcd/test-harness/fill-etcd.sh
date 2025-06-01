#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

. etcd-perftest-config

FILL_GB=5
dt=$(date +"%Y-%m-%d-%H-%M")

# 1462 is the number of puts required to add ~1 MB to the db
TOTAL_PUTS=$((FILL_GB*1024*1462))

echo "Start: ${dt}"
etcdctl --endpoints=${ETCD_ENDPOINTS} endpoint status -w table
etcd-benchmark put --endpoints ${ETCD_ENDPOINTS} --clients 20 --conns 20 --key-size 100 --val-size 412 --total ${TOTAL_PUTS}
etcdctl --endpoints=${ETCD_ENDPOINTS} endpoint status -w table
echo "End: $(date +'%Y-%m-%d-%H-%M')"
