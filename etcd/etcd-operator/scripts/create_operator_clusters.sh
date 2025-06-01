#!/bin/bash -ex
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# The main loop for creating etcd-operator clusters.
# It is assumed that this script is called from a jenkins job that sets up
# the required environment and the necessary exported variables.
# https://alchemy-testing-jenkins.swg-devops.com/job/Armada-Performance/job/Etcd-Operator/job/CreateEtcdOperatorClusters/

function check_existence {
    if [[ -z $2 ]]; then
        echo "ERROR: $1 environment variable must be defined"
        exit 1
    fi
}

# Make sure this is set
if [[ -z $CLUSTER_NAME_PREFIX ]]; then
    echo "ERROR: CLUSTER_NAME_PREFIX environment variable must be define with non-zero length string"
    exit 1
fi

check_existence WORKSPACE $WORKSPACE

if [[ ! -f ${WORKSPACE}/etcd-operator/example/tls/example-tls-cluster.yaml ]]; then
    echo "ERROR: Environment isn't setup with \${WORKSPACE}/etcd-operator/example/tls/example-tls-cluster.yaml"
    exit 1
fi

if [[ -z $COUNT ]]; then
    export COUNT=1
fi

if [[ -z $THREADS ]]; then
    export THREADS=1
fi

CLUSTERS_PER_THREAD=$COUNT
if [[ $THREADS -gt $COUNT ]]; then
    THREADS=$COUNT
fi

if [[ $THREADS -gt 1 ]]; then
    CLUSTERS_PER_THREAD=$((COUNT/THREADS))
    PLUS_ONE_THREADS=$((COUNT-CLUSTERS_PER_THREAD*THREADS))
fi

PLUS_ONE=1
for (( THREAD=1; THREAD<=$THREADS; THREAD=THREAD+1 )); do
    echo "Initiating creation of $CLUSTERS_PER_THREAD etcd-operator clusters in thread $THREAD."
    if [[ $THREAD -gt $PLUS_ONE_THREADS ]]; then
        PLUS_ONE=0
    fi
    ${WORKSPACE}/armada-performance/etcd/etcd-operator/scripts/create_operator_cluster_thread.sh $THREAD $((CLUSTERS_PER_THREAD+PLUS_ONE)) ${CLUSTER_NAME_PREFIX}${THREAD}-  >> EtcdOperatorClusterCreateOut_thread${THREAD}.txt 2>&1 &
done

jobs
echo "Waiting on $THREADS threads to complete"
wait
