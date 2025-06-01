#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# Script to print endpoint status (Which includes the DB Size) for all members of a carrier etcd.
# If a parameter is passed in then it will also Defrag the members, printing the status before and after defrag.
# This should be run from the carrier master.

downloadEtcdctl() {
    ETCD_VER=v3.4.0
    GITHUB_URL=https://github.com/etcd-io/etcd/releases/download
    DOWNLOAD_URL=${GITHUB_URL}
    rm -f etcd/etcd-${ETCD_VER}-linux-amd64.tar.gz
    rm -rf etcd/etcd-download-test && mkdir -p etcd/etcd-download-test

    curl -L ${DOWNLOAD_URL}/${ETCD_VER}/etcd-${ETCD_VER}-linux-amd64.tar.gz -o etcd/etcd-${ETCD_VER}-linux-amd64.tar.gz
    tar xzvf etcd/etcd-${ETCD_VER}-linux-amd64.tar.gz -C etcd --strip-components=1
    rm -f etcd/etcd-${ETCD_VER}-linux-amd64.tar.gz
    rm -f etcd/etcd
}

OIFS=$IFS
IFS=$'\n'

FILE=etcd/etcdctl
if [ ! -f "$FILE" ]; then
    echo "$FILE does not exist - will download it"
    downloadEtcdctl
fi

defrag=$1

# Use the etcd on localhost to get the list of etcd member endpoints
member_list=$(${FILE} --endpoints https://127.0.0.1:4001 --cert /etc/kubernetes/cert/etcd.pem --key /etc/kubernetes/cert/etcd-key.pem --cacert /etc/kubernetes/cert/ca.pem member list | awk '{print $5}' | tr -d ',')

# Now list the endpoint status for each member
echo "Member status before defrag:"
for member in $member_list; do
    ${FILE} --endpoints ${member} --cert /etc/kubernetes/cert/etcd.pem --key /etc/kubernetes/cert/etcd-key.pem --cacert /etc/kubernetes/cert/ca.pem endpoint status
done
if [[ -n $defrag ]]; then
    for member in $member_list; do
        ${FILE} --endpoints ${member} --cert /etc/kubernetes/cert/etcd.pem --key /etc/kubernetes/cert/etcd-key.pem --cacert /etc/kubernetes/cert/ca.pem defrag
        sleep 10
    done
    echo "Member status after defrag:"
    for member in $member_list; do
        ${FILE} --endpoints ${member} --cert /etc/kubernetes/cert/etcd.pem --key /etc/kubernetes/cert/etcd-key.pem --cacert /etc/kubernetes/cert/ca.pem endpoint status
    done
fi

IFS=$OIFS
