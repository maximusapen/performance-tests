#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

if [[ $# -ne 1 ]]; then
    echo "Usage: `basename $0` <endpoints>"
    exit 1
fi

STATSFILE=results/etcdStats.txt

ETCDCTRL=/opt/bin/etcdctl

ENDPOINTS="$1"

THEDATE=$(date)

export ETCDCTL_API=3

echo '--- Dumping stats to file at ' $THEDATE ' ---' >> $STATSFILE

$ETCDCTRL $ETCDCREDS --endpoints $ENDPOINTS alarm list >> $STATSFILE
$ETCDCTRL $ETCDCREDS --endpoints $ENDPOINTS --write-out=table endpoint status >> $STATSFILE

array=(${ENDPOINTS//,/ })
for i in "${!array[@]}"
do
   echo 'Endpoint: ' ${array[i]} >> $STATSFILE
   curl -L ${array[i]}/metrics >> $STATSFILE
done
