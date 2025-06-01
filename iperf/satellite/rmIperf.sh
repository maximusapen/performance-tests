#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# Remove all iperf clients or servers

if [[ $# -lt 2 ]]; then
    echo
    echo "Usage: ./rmIperf.sh <client | server> <number_of_instance>"
    echo
    exit 1
fi

# iperfType is client or server
iperfType=$1
# Number of client or server pod
maxPod=$2

if [[ ${iperfType} == "client" ]]; then
    if [[ -z ${CLIENT_KUBECONFIG} ]]; then
        echo "You need to export CLIENT_KUBECONFIG"
        exit 1
    fi
    export KUBECONFIG=${CLIENT_KUBECONFIG}
    echo Using CLIENT_KUBECONFIG ${CLIENT_KUBECONFIG}
elif [[ ${iperfType} == "server" ]]; then
    if [[ -z ${SERVER_KUBECONFIG} ]]; then
        echo "You need to export SERVER_KUBECONFIG"
        exit 1
    fi
    export KUBECONFIG=${SERVER_KUBECONFIG}
    echo Using SERVER_KUBECONFIG ${SERVER_KUBECONFIG}
else
    echo "Usage: ./rmIperf.sh <client | server> <number_of_instance>"
    echo
    exit 1
fi

for i in $(seq 1 ${maxPod}); do
    helm uninstall iperf${iperfType}-$i
done
