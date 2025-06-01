#!/bin/bash

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
