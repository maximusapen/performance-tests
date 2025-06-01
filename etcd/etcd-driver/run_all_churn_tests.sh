#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
dir=`dirname $0`

# Arguments to run_churn_test.sh are:
#  <etcd-cluster-name (etcd-slnfs or etcd)> <test-duration> <watches-per-level> <pattern> <churn-val-rate> <churn-rate> <churn-level> <churn-pct> <val-spec> <get-level> <get-rate> <comment>

# Carrier Kubernetes tests
$dir/run_churn_test.sh etcd-slnfs 15 1,2,6,2,5000,100 "/prefix/%level1-%06d[2]/%level2-%04d[6]/%level3-%04d[2]/%level4-%040d[5000]/%leaf5-%06d[1];[0-9]{1,10}" 360000 36000 4 10 1000,10000 4 360000 "Standard_Carrier_kube_test"
sleep 120
$dir/run_churn_test.sh etcd-slnfs 15 2,4,12,4,10000,200 "/prefix/%level1-%06d[2]/%level2-%04d[6]/%level3-%04d[2]/%level4-%040d[5000]/%leaf5-%06d[1];[0-9]{1,10}" 360000 36000 4 10 1000,10000 4 360000 "Carrier_kube_test_double_watchers"
sleep 120
$dir/run_churn_test.sh etcd-slnfs 20 1,2,6,2,5000,100 "/prefix/%level1-%06d[2]/%level2-%04d[6]/%level3-%04d[2]/%level4-%040d[10000]/%leaf5-%06d[1];[0-9]{1,10}" 360000 36000 4 10 1000,10000 4 360000 "Carrier_kube_test_double_keys"
sleep 120
$dir/run_churn_test.sh etcd-slnfs 15 1,2,6,2,5000,100 "/prefix/%level1-%06d[2]/%level2-%04d[6]/%level3-%04d[2]/%level4-%040d[5000]/%leaf5-%06d[1];[0-9]{1,10}" 0 0 4 10 1000,10000 4 0 "Carrier_kube_test_no_rate_limit"

# Armada etcd tests
sleep 120
$dir/run_churn_test.sh etcd-slnfs 20 1,2,1,6000,2,25 "/prefix00/%level1-%02d[2]/%level2-%02d[1]/%level3-%024d[6000]/%level4-%050d[2]/%leaf5-%010d[25];[0-9]{1,10}" 360000 36000 4 10 10,20 4 360000 "Standard_Aramada_etcd_test"

sleep 120
$dir/run_churn_test.sh etcd-slnfs 20 2,4,2,12000,4,50 "/prefix00/%level1-%02d[2]/%level2-%02d[1]/%level3-%024d[6000]/%level4-%050d[2]/%leaf5-%010d[25];[0-9]{1,10}" 360000 36000 4 10 10,20 4 360000 "Aramada_etcd_test_double_watchers"

sleep 120
$dir/run_churn_test.sh etcd-slnfs 20 1,2,1,6000,2,25 "/prefix00/%level1-%02d[2]/%level2-%02d[1]/%level3-%024d[12000]/%level4-%050d[2]/%leaf5-%010d[25];[0-9]{1,10}" 360000 36000 4 10 10,20 4 360000 "Aramada_etcd_test_double_keys"

sleep 120
$dir/run_churn_test.sh etcd-slnfs 20 1,2,1,6000,2,25 "/prefix00/%level1-%02d[2]/%level2-%02d[1]/%level3-%024d[6000]/%level4-%050d[2]/%leaf5-%010d[25];[0-9]{1,10}" 0 0 4 10 10,20 4 0 "Aramada_etcd_test_no_rate_limit"
