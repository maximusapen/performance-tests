#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright Maximus Apen, 2025 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Run a suite of iperf tests

# WARNING: Don't use the '-f' flag of run_iperf.sh

IPERF_DIR=/performance/armada-perf/iperf/bin
pushd ${IPERF_DIR}
# Baselines
export CLIENT_KUBECONFIG=/performance/config/iperfClient/kube-config-dal09-iperfClient.yml
export SERVER_KUBECONFIG=/performance/config/iperfServer/kube-config-dal09-iperfServer.yml

#echo "TEST: Baseline Load Balancer"
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 1 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 1 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 1 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 1 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 1 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 1 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 4 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 4 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 4 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 4 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 4 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 4 -l -o -P 4

${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -l -o -P 4

#Node Ports
#echo "TEST: Baseline Node Ports"
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 1 -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 1 -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 1 -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 1 -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 1 -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 1 -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 4 -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 4 -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 4 -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 4 -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 4 -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 4 -o -P 4

# Outbound
export CLIENT_KUBECONFIG=/performance/config/privClusterJuly23b/kube-config-dal09-privClusterJuly23b.yml
export SERVER_KUBECONFIG=/performance/config/iperfServer/kube-config-dal09-iperfServer.yml

#Node Ports
#echo "TEST: Outbound Node Ports"
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 3 -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 3 -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 3 -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 3 -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 3 -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 3 -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 6 -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 6 -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 6 -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 6 -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 6 -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 6 -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 12 -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 12 -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 12 -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 12 -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 12 -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 12 -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -o -P 4

#Load Balancer
#echo "TEST: Outbound LoadBalancer"
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 3 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 3 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 3 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 3 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 3 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 3 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 6 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 6 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 6 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 6 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 6 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 6 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 12 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 12 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 12 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 12 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 12 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 12 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -l -o -P 4

# Inbound Load Balancer
export CLIENT_KUBECONFIG=/performance/config/iperfClient/kube-config-dal09-iperfClient.yml
export SERVER_KUBECONFIG=/performance/config/privClusterJuly23b/kube-config-dal09-privClusterJuly23b.yml

echo "TEST: Inbound LoadBalancer"
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 3 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 3 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 3 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 3 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 3 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 3 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 6 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 6 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 6 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 6 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 6 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 6 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 12 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 12 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 12 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 12 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 12 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 12 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -l -o -P 4

#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 18 -l -o -P 4

# 17 pairs
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 17 -l -P 4
#${IPERF_DIR}/run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage5 --concurrency 17 -l -o -P 4

popd
