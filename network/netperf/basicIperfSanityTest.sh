#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright Maximus Apen, 2025 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Runs brief iperf3 test to determine, via host network usage graphs, which firewall is being used by which netperf-pod
# Assumes that KUBECONFIG is set to where the previously created netperf-pods are running

# Set ip and port of iperf3 in server mode
server_ip=169.44.1.224
server_port=30522
namespace=iperf
testduration=120
for pod in `kubectl -n ${namespace} get pods | grep 'netperf-pod[0-9]*' | awk '{print $1}'`; do
    echo "$(date) Running iperf on $pod"
    kubectl -n ${namespace} exec ${pod} -- iperf3 -t ${testduration} -J -c ${server_ip} -p ${server_port} > $pod.iperf.json

    # Sleep a bit to produce a gap in host statistics graph
    sleep 30
done
