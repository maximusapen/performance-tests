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


export ETCDCTL_API=3

$ETCDCTRL $ETCDCREDS --endpoints $ENDPOINTS put "/test1/end" "true"
