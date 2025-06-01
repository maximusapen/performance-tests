#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# Run specified number of iperfclient connecting directt tto same namber of iperfserver running
# in parallel for duration in seconds.

# Set up client and sever kubeconfig as follows before running this script
# export CLIENT_KUBECONFIG=<full path to file>/kube-config-<location>-<<name of your client cluster>.yml
# export SERVER_KUBECONFIG=<full path to file>/kube-config-<location>-<<name of your server cluster>.yml

if [[ $# -lt 2 ]]; then
    echo
    echo "Usage: ./runClientDirect.sh <number_of_clients> <duration_in_seconds> "
    echo
    exit 1
fi

declare -i nClient=$1
declare -i duration=$2

source envFile

if [[ -z ${SERVER_KUBECONFIG} || -z ${CLIENT_KUBECONFIG} ]]; then
    echo "You need to export SERVER_KUBECONFIG and CLIENT_KUBECONFIG"
    exit 1
fi

echo Using SERVER_KUBECONFIG ${SERVER_KUBECONFIG}
echo Using CLIENT_KUBECONFIG ${CLIENT_KUBECONFIG}

export KUBECONFIG=${SERVER_KUBECONFIG}
kubectl get pod -o wide >/tmp/iperfServers
kubectl get service | grep NodePort >/tmp/iperfServices
kubectl get node -o wide >/tmp/iperfserverNodes
echo "IPerf servers found:"
cat /tmp/iperfServers
echo "IPerf services found:"
cat /tmp/iperfServices
echo "Nodes in cluster:"
cat /tmp/iperfserverNodes

export KUBECONFIG=${CLIENT_KUBECONFIG}
cd ${perf_dir}/iperf/bin
for i in $(seq 1 ${nClient}); do
    echo Processing client $i
    iperfServer=$(grep iperfserver-$i-deployment /tmp/iperfServers)
    host=$(echo $iperfServer | awk '{print $7}')
    dest_host=$(grep ${host} /tmp/iperfserverNodes | awk '{print $7}')
    echo host: $host dest_host: $dest_host
    echo
    echo iperfclient.sh --id $i --address $dest_host -t $duration -J
    ./iperfclient.sh --id $i --address $dest_host -t $duration -J
done
