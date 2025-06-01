#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Set up your KUBECONFIG before running this script
#export KUBECONFIG=<cruiser kube config file>

# Parameter - scale to the number of replicas
scale=$1

# By default, use pod httpperf in namespace httpperf1
# You would have created httpperf1 namespace and deployed httpperf in the namespace
# as follows:
#     ./createPodRegistry 1 1
#     ./createtestpods.sh 1 1 30000

namespace=httpperf1
pod=httpperf


kubectl scale deployment "${pod}" -n "${namespace}" --replicas "${scale}"
