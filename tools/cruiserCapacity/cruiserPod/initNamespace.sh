#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018, 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Set up httpperf1 namespace

# Set up your KUBECONFIG before running this script
#export KUBECONFIG=<cruiser kube config file>

# Assumed running in performance client.  Modify PerfBin if necessary
PerfBin=/performance/armada-perf/automation/bin

namespace=httpperf1
echo creating $namespace
echo Setting up Registery Access for namespace $namespace

echo "$(date +%Y%m%d-%H%M%S) start creating registry secret for $namespace"
$PerfBin/setupRegistryAccess.sh $namespace
echo "$(date +%Y%m%d-%H%M%S) registry secret created for $namespace"
