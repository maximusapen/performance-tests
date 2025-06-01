#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
dir=`dirname $0`

if [[ $# -lt 12 ]]; then
    echo "Usage: `basename $0`  <etcd-cluster-name (etcd-slnfs or etcd)> <test-duration (mins)> <watches-per-level> <pattern> <churn-val-rate> <churn-rate> <churn-level> <churn-pct> <val-spec> <get-level> <get-rate> <comment> {<extra-get-args>}"
    exit 1
fi
ETCDCLUSTER=$1
shift
TESTDURATION=$1
shift
WATCHES=$1
shift
PATTERN=$1
shift
CHURNVALRATE=$1
shift
CHURNRATE=$1
shift
CHURNLEVEL=$1
shift
CHURNPCT=$1
shift
VALSPEC=$1
shift
GETLEVEL=$1
shift
GETRATE=$1
shift
COMMENT=$1
EXTRA_ARGS=$2
ETCDCTRL=/opt/bin/etcdctl

# This will set ENDPOINTS which is the endpoints of the pods
. /perftest/etcd/scripts/getEndpoints.sh "$ETCDCLUSTER"

# This will set SERVICE_ENDPOINTS which will be the node ports
# The client will use these so that it continues to work after etcd restarts
. /perftest/etcd/scripts/getServiceEndpoints.sh
export ETCDCTL_API=3
THEDATE=$(date)
echo '--- Starting test at ' $THEDATE
echo 'Deleting etcd contents'
$ETCDCTRL $ETCDCREDS --endpoints $ENDPOINTS del "/" --prefix
$dir/etcdCompressDb.sh $ENDPOINTS

echo 'Starting watchers'
$dir/runWatcher.sh $SERVICE_ENDPOINTS $WATCHES $PATTERN $COMMENT >> results/WatcherOutput.txt 2>&1 &
sleep 10

echo 'Dumping stats at start of test'
$dir/etcdDumpStats.sh $ENDPOINTS

echo 'Starting GetDBSize thread'
$dir/getDBsize.sh $ENDPOINTS $TESTDURATION &

echo 'Starting Churn'
$dir/runChurn.sh $SERVICE_ENDPOINTS $PATTERN $CHURNVALRATE $CHURNRATE $CHURNLEVEL $CHURNPCT $VALSPEC $GETLEVEL $GETRATE $COMMENT "$EXTRA_ARGS" >> results/churnOutput.txt 2>&1 &

echo 'Waiting for ' $TESTDURATION 'm before stopping test'
sleep $(($TESTDURATION))m

echo 'Sending test end signal'
$dir/endTest.sh $ENDPOINTS
sleep 10

echo 'Dumping stats at end of test'
$dir/etcdDumpStats.sh $ENDPOINTS

$dir/etcdCompressDb.sh $ENDPOINTS
$dir/etcdDumpStats.sh $ENDPOINTS

kubectl logs etcd-slnfs-0 > results/etcd-slnfs-0-`date "+%Y%m%d-%H%M"`.log 2>&1
kubectl logs etcd-slnfs-1 > results/etcd-slnfs-1-`date "+%Y%m%d-%H%M"`.log 2>&1
kubectl logs etcd-slnfs-2 > results/etcd-slnfs-2-`date "+%Y%m%d-%H%M"`.log 2>&1
