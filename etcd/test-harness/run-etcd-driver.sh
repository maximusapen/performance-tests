#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

. etcd-perftest-config

dt=$(date +"%Y-%m-%d-%H-%M")
echo "Start: ${dt} etcd-driver"
#      --cert=/etc/etcdtls/operator/etcd-tls/etcd-client.crt
#      --key=/etc/etcdtls/operator/etcd-tls/etcd-client.key
#      --cacert=/etc/etcdtls/operator/etcd-tls/etcd-client-ca.crt
etcd-driver pattern --endpoints=${ETCD_ENDPOINTS} \
      --conns=100 \
      --clients=100 \
      --pattern='/prefix/%level1-%06d[2]/%level2-%04d[6]/%level3-%04d[2]/%level4-%040d[80]/%leaf5-%06d[1];[0-9]{1,10}' \
      --csv-file=churn_results.csv \
      --churn-val-rate=0 \
      --churn-level-rate=0 \
      --churn-level=4 \
      --churn-level-pct=10 \
      --test-end-key=/test1/end \
      --val-spec=100,1000 \
      --get-level=4 \
      --get-rate=10040000 \
      --stats-interval=20
echo "End: $(date +'%Y-%m-%d-%H-%M')"
#      --churn-val-rate=36000 \
#      --churn-level-rate=3600 \
./defrag-etcd.sh compact
