#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

if [[ $# -ne 1 ]]; then
    echo "Usage: `basename $0` <endpoints>"
    exit 1
fi

ETCDCTRL=/opt/bin/etcdctl

ENDPOINTS="$1"
SLEEP=30

ETCDCTL_API=3

$ETCDCTRL $ETCDCREDS --endpoints $ENDPOINTS alarm list
$ETCDCTRL $ETCDCREDS --endpoints $ENDPOINTS --write-out=table endpoint status

rev=$($ETCDCTRL $ETCDCREDS --endpoints $ENDPOINTS endpoint status --write-out="json" | egrep -o '"revision":[0-9]*' | egrep -o '[0-9]+' | head -1)
time $ETCDCTRL $ETCDCREDS --endpoints $ENDPOINTS compact $rev
if [[ $? -eq 0 ]]; then
    sleep $SLEEP

    array=(${ENDPOINTS//,/ })
    for i in "${!array[@]}"
    do
        time $ETCDCTRL $ETCDCREDS --endpoints ${array[i]} defrag --command-timeout 60s
    done
    sleep $SLEEP
fi

$ETCDCTRL $ETCDCREDS --endpoints $ENDPOINTS --write-out=table endpoint status
$ETCDCTRL $ETCDCREDS --endpoints $ENDPOINTS alarm disarm
