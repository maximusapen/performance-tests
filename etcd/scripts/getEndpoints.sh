#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

FILTER=""

if [[ $# -eq 1 ]]; then
	FILTER="$1"
fi

ENDPOINTS=""
for pod in `kubectl get pods | grep etcd | cut -d" " -f1 | grep "$FILTER\-[0-9]"`; do
	if [[ -z $FILTER ]]; then
		echo $pod
	fi
	ip=`kubectl describe pods $pod| grep "^IP" | sed -e "s/^IP:[\t]*//g"`
	if [[ -z $ENDPOINTS ]]; then
		ENDPOINTS="${ip}:2379"
	else
		ENDPOINTS="${ENDPOINTS},${ip}:2379"
	fi
done
echo "$ENDPOINTS"

if [[ -z $FILTER ]]; then
	etcdctl --endpoints $ENDPOINTS --write-out=table endpoint status
fi
