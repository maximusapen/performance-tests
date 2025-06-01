#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019, 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Add secrets to specified cluster # in parallel
#
# Run after logging into IKS test carrier and after generate_secrets_4.sh

source ../test_conf.sh

if [ $# -eq 2 ]; then
    echo "Override secretStart from $secretStart to $1"
    echo "Override secretEnd from $secretEnd to $2"
    secretStart=$1
    secretEnd=$2
fi

echo Run ${numThread} threads with ${batch} clusters in each thread

rm add_secrets*.log

start=${clusterStart}
for i in $(seq 1 ${numThread}); do
    end=$((${start} + ${batch} - 1))
    logFile="add_secrets${i}.log"
    echo Starting thread $i: cluster ${start} to ${end}.
    ./add_secrets.sh ${start} ${end} ${secretStart} ${secretEnd} >${logFile} 2>&1 &
    start=$((${end} + 1))
done

SECONDS=0
nGetProcess=$(ps -ef | grep add_secrets.sh | grep -v grep | wc -l)
while [[ ${nGetProcess} != "0" ]]; do
    echo "Number of add_secrets processes: ${nGetProcess}"
    nGetProcess=$(ps -ef | grep add_secrets.sh | grep -v grep | wc -l)
    sleep 30
done
duration=${SECONDS}
echo "Time to add secret(s) for all clusters: $(($duration / 60)) minutes and $(($duration % 60)) seconds (${duration} seconds)."

echo "Check errors in logs."
nErrors=$(grep Error add_secrets*.log | wc -l)
echo "Number of Errors when adding secrets: ${nErrors})"
grep Error add_secrets*.log

echo "Check output in secret/add_secrets*.log"
