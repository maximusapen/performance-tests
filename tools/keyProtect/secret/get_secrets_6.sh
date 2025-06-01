#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019, 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Get secrets from clusters in parallel
#

source ../test_conf.sh

echo Run ${numThread} threads with ${batch} clusters in each thread

set +e
rm get_secrets*.log
set -e

start=${clusterStart}
for i in $(seq 1 ${numThread}); do
    end=$((${start} + ${batch} - 1))
    echo Starting thread $i: cluster ${start} to ${end}
    ./get_secrets.sh ${start} ${end} >get_secrets${i}.log 2>&1 &
    start=$((${end} + 1))
done
echo "Check output in secret/get_secrets*.log"

SECONDS=0
nGetProcess=$(ps -ef | grep get_secrets.sh | grep -v grep | wc -l)
while [[ ${nGetProcess} != "0" ]]; do
    echo "Number of get_secrets processes: ${nGetProcess}"
    nGetProcess=$(ps -ef | grep get_secrets.sh | grep -v grep | wc -l)
    sleep 30
done
duration=${SECONDS}

for i in $(seq 1 ${numThread}); do
    echo
    cat get_secrets${i}.log
done

echo "Time to get all secrets for all clusters: $(($duration / 60)) minutes and $(($duration % 60)) seconds (${duration} seconds)."
echo "Number of TimeoutError: $(grep Timeout get_secrets*.log | wc -l)"
echo "Number of InternalError: $(grep InternalError get_secrets*.log | wc -l)"
