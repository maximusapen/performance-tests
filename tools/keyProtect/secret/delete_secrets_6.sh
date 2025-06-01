#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019, 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Delete secrets from specified cluster # in parallel

source ../test_conf.sh

if [ $# -eq 2 ]; then
    echo "Override secretStart from $secretStart to $1"
    echo "Override secretEnd from $secretEnd to $2"
    secretStart=$1
    secretEnd=$2
fi

echo Run ${numThread} threads with ${batch} clusters in each thread

set +e
rm delete_secrets*.log
set -e

start=${clusterStart}
for i in $(seq 1 ${numThread}); do
    end=$((${start} + ${batch} - 1))
    logFile="delete_secrets${i}.log"
    echo Starting thread $i: cluster ${start} to ${end}.
    ./delete_secrets.sh ${start} ${end} ${secretStart} ${secretEnd} >${logFile} 2>&1 &
    start=$((${end} + 1))
done
echo "Check output in secret/delete_secrets*.log"
