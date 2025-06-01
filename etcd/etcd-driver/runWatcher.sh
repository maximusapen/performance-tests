#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
dir=`dirname $0`

if [[ $# -ne 4 ]]; then
    echo "Usage: `basename $0` <endpoints> <watches-per-level> <pattern> <comment>"
    exit 1
fi

ENDPOINTS=$1
shift
WATCHES=$1
shift
PATTERN=$1
shift
COMMENT="$1"

$dir/etcd-driver watch-tree $ETCDCREDS --endpoints $ENDPOINTS --conns=3 --clients=3 --pattern $PATTERN --watch-counts-per-level $WATCHES --with-prefix --csv-file results/watch_results.csv --file-comment $COMMENT --test-end-key /test1/endwatch --stats-interval=60 >> results/watchOutdm.txt 2>&1
