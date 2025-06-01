#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

if [[ $# != 1 ]]; then
    echo "Usage: etcdAddAuthentication.sh <endpoints>"
    exit 1
fi

ENDPOINTS=$1

echo "Default password for user should be <username>pw"
etcdctl --endpoints $ENDPOINTS role add root
etcdctl --endpoints $ENDPOINTS role add guest
etcdctl --endpoints $ENDPOINTS user add root
etcdctl --endpoints $ENDPOINTS user grant-role root root
etcdctl --endpoints $ENDPOINTS user add etcddriver
etcdctl --endpoints $ENDPOINTS role add etcddriver
etcdctl --endpoints $ENDPOINTS user grant-role etcddriver  etcddriver
etcdctl --endpoints $ENDPOINTS --user root:rootpw role grant-permission  etcddriver readwrite /etcdbm /etcdbm0
etcdctl --endpoints $ENDPOINTS auth enable
