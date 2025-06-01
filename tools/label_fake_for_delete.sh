#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

NAMESPACE=kubx-masters
CARRIER=carrier1
DATACENTER=dal09
if [[ $CARRIER == "carrier3 || $CARRIER == carrier5 ]]; then
    DATACENTER=dal10
fi
#NUM_WORKERS=75
# TODO Best to take total number of cruisers to keep and deal with the complexity that
#      there will be an inbalance of cruisers on nodes

USAGE="Usage: debel_fake_for_delete.sh <number of cruisers to be maintianed on each node>"

if [[ $# -ne 1 ]]; then
    echo $USAGE
	exit 1
fi

WORKERS_PER_NODE=$1

echo "Getting list of cruisers"
kubectl --no-headers -n kubx-masters get pods -o wide > /tmp/fake_cruisers.tmp

# Get cruiser names
cut -d" " -f1 /tmp/fake_cruisers.tmp | sort > /tmp/all_cruisers.tmp

# Get list of cruisers node/names sorted by node
cat /tmp/fake_cruisers.tmp | tr -s ' ' | cut -d ' ' -f 1,7 | awk '{print $2, $1}' | sort > /tmp/cruisers.by.node.tmp

echo "Generating a list of cruisers to keep"
for i in `grep stage-${DATACENTER}-${CARRIER}-worker- /etc/hosts | awk '{print $1}'`; do 
	for j in `grep $i /tmp/cruisers.by.node.tmp | head -${WORKERS_PER_NODE} | cut -d" " -f2`; do 
		echo $j
	done
done | sort > /tmp/keep.1794.tmp

echo "Clearing label on all cruisers"
kubectl -n $NAMESPACE label pod fake-cruiser-delete- --all 1>/dev/null 2>&1

for i in `cat /tmp/keep.1794.tmp`; do 
	grep $i  /tmp/fake_cruisers.tmp
done > /tmp/keepbynode.tmp

# Get a list of cruisers to delete
diff /tmp/all_cruisers.tmp /tmp/keep.1794.tmp |grep master | sed -e "s/< //g" > /tmp/dump.tmp

echo "Label cruisers to be deleted"
for i in `cat /tmp/dump.tmp`; do
    kubectl -n $NAMESPACE label pod $i fake-cruiser-delete=true
done

# Get a list to dump by node
for i in `cat /tmp/dump.tmp`; do 
	grep $i  /tmp/fake_cruisers.tmp
done > /tmp/dumpbynode.tmp

echo "Total cruisers to be preserved is `wc -l /tmp/keep.1794.tmp`"

echo "Balance of cruisers across nodes after deletes"
cat /tmp/keepbynode.tmp | tr -s ' ' | cut -d ' ' -f 7 | sort | uniq -c
