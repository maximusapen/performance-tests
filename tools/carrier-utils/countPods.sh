#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# Script that will repeatedly count the number of different types of pods on each node.
# It requires access to kubectl and a /etc/hosts file that contains entries for the hosts on the carrier.
#
if [[ $# -ne 2 ]]; then
    echo "Usage: `basename $0` <interval> <repeats>"
    exit 1
fi
interval=$1
repeats=$2

for (( i=0; i<repeats; i=i+1 )); do
    echo "Date, Hostname, IP, master pods running, master pods total, openvpn pods running, openvpn pods total, clust-update pods running, clust-update pods total, etcd pods running, etcd pods total, all pods running, all pods total"
    DATE=$(date --utc +%FT%TZ)
    allpods=$(kubectl get pods --all-namespaces -o=wide)
    mpods=$(echo "$allpods" | grep kubx-masters | grep master-)
    cupods=$(echo "$allpods" | grep kubx-masters | grep cluster-updater)
    ovpods=$(echo "$allpods" | grep kubx-masters | grep openvpnserver)
    eopods=$(echo "$allpods" | grep kubx-etcd)
    allpods=$(kubectl get pods --all-namespaces -o=wide)
    result=""
    for n in `kubectl get nodes --no-headers| awk '{print $1}'`; do
        ov_running=$(echo "$ovpods" | grep -w "$n" | grep Running | wc -l)
        ov_total=$(echo "$ovpods" | grep -w "$n" | wc -l)
        mrunning=$(echo "$mpods" | grep -w "$n" | grep Running | wc -l)
        mtotal=$(echo "$mpods" | grep -w "$n" | wc -l)
        cu_running=$(echo "$cupods" | grep -w "$n" | grep Running | wc -l)
        cu_total=$(echo "$cupods" | grep -w "$n" | wc -l)
        eo_running=$(echo "$eopods" | grep -w "$n" | grep Running | wc -l)
        eo_total=$(echo "$eopods" | grep -w "$n" | wc -l)
        all_running=$(echo "$allpods" | grep -w "$n" | grep Running | wc -l)
        all_total=$(echo "$allpods" | grep -w "$n" | wc -l)
        hostname=$(cat /etc/hosts | grep -a -m 1 $n | awk '{print $2}' | cut -d$'.' -f1)
        if [[ -z $hostname ]]; then
          hostname=$n
        fi

        result+="$DATE,$hostname,$n,$mrunning,$mtotal,$ov_running,$ov_total,$cu_running,$cu_total,$eo_running,$eo_total,$all_running,$all_total"$'\n'
    done
    echo "$result" | sort -k 2 -t ','
    sleep $interval
done
