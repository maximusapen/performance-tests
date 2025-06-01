#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

. etcd-perftest-config

dt=$(date +"%Y-%m-%d-%H-%M")

CLIENTS=100
echo "Start: ${dt} etcd-benchmark put test with ${CLIENTS} clients"
etcdctl --endpoints=${ETCD_ENDPOINTS} endpoint status -w table
./etcd-benchmark put --endpoints ${ETCD_ENDPOINTS} --clients ${CLIENTS} --conns 10 --key-size=200 --total=500000 --val-size=500 --key-space-size=100000 --csv-file=etcdBenchmarksResults.csv --file-comment="Puts" 
#--put-rate 5000
etcdctl --endpoints=${ETCD_ENDPOINTS} endpoint status -w table
echo "End: $(date +'%Y-%m-%d-%H-%M')"
