#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Deletes all the netperf-pod# clients and associated services

namespace=iperf
lb_postfix=""

if [[ $# -ge 1 && $1 == "true" ]]; then
    lb_postfix="-lb"
fi


for pod in `kubectl -n ${namespace} get pods | grep "netperf-pod[0-9]*${lb_postfix}" | awk '{print $1}'`; do
    kubectl -n ${namespace} delete pod $pod
done

kubectl -n ${namespace} get pods
