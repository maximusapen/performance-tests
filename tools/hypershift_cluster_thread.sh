#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
THREAD=$1
CLUSTER_PREFIX=$2
TUGBOAT_TYPE=$3
BOM_VERSION=$4
NUM_CLUSTERS=$5

export KUBECONFIG=${HYPERSHIFT_ROOT}/openshift-test-management-cluster-kubeconfig

cd ${HYPERSHIFT_ROOT}/repo${THREAD}/armada-openshift-master
source .venv/bin/activate

python --version

echo "Tugboat Config: ${KUBECONFIG}"
echo "Tugboat Type: ${TUGBOAT_TYPE}"

export BOM_FILE_PATH=../armada-ansible/common/bom/next/openshift-target-bom-${BOM_VERSION}.yml
export TEST_MASTER_TARGET_BOM=${HYPERSHIFT_ROOT}/repo${THREAD}/armada-ansible/common/bom/next/openshift-target-bom-${BOM_VERSION}.yml

export MANAGED_CLUSTER_TYPE=${TUGBOAT_TYPE}
export CLUSTER_DEFAULT_PROVIDER=upi
export HYPERSHIFT=true

echo
echo "Preparing environment"
molecule prepare --force -s ${TUGBOAT_TYPE}-cluster
echo

for ((CLUSTER = 1; CLUSTER <= ${NUM_CLUSTERS}; CLUSTER = CLUSTER + 1)); do
    export TEST_CLUSTER_ID="${CLUSTER_PREFIX}-${THREAD}-${CLUSTER}"

    echo
    printf "%s - Creating cluster : '%s'\n" "$(date +%FT%T)" "${TEST_CLUSTER_ID}"

    molecule converge -s ${TUGBOAT_TYPE}-cluster
done
