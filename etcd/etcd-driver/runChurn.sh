#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
dir=`dirname $0`

if [[ $# -lt 10 ]]; then
    echo "Usage: `basename $0` <endpoints> <pattern> <churn-val-rate> <churn-rate> <churn-level> <churn-pct> <val-spec> <get-level> <get-rate> <comment> {<extra-get-args>}"
    exit 1
fi

ENDPOINTS=$1
shift
PATTERN=$1
shift
CHURNVALRATE=$1
shift
CHURNRATE=$1
shift
CHURNLVL=$1
shift
CHURNPCT=$1
shift
VALSPEC=$1
shift
GETLVL=$1
shift
GETRATE=$1
shift
COMMENT="$1"
EXTRA_ARGS=$2

echo "Churning keys at `date "+%Y%d%m-%H%M.%S"`, $COMMENT"
$dir/etcd-driver pattern $ETCDCREDS --endpoints $ENDPOINTS --conns=9 --clients=9 --pattern $PATTERN --csv-file results/churn_results.csv --churn-val-rate $CHURNVALRATE --churn-level-rate $CHURNRATE --churn-level $CHURNLVL --churn-level-pct $CHURNPCT --test-end-key /test1/end --val-spec $VALSPEC --get-level $GETLVL --get-rate $GETRATE --file-comment $COMMENT --stats-interval=60 $EXTRA_ARGS >>results/churnOutdm.txt 2>&1
