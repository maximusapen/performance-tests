#!/bin/bash
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
