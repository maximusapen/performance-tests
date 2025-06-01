#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# Script for creating multiple hypershift based ROKS hosted clusters.
#
# Required Environment
# ====================
#
# Ensure the <HYPERSHIFT_ROOT> folder is created in a suitable location of your choice.
# e.g. mkdir $HOME/hypershift
#
# One time perf client setup
# --------------------------
#   1. Install python 3,8 (if not already installed. N.B. Ubuntu 18 comes with 3.6.9)
#     sudo apt-get install python3.8 python3.8-dev python3.8-distutils python3.8-venv
#
#   2. Ensure correct versions of kubectl, calicoctl, cfssl, cfssljson are gpg are installed
#     cfssl: See https://github.com/cloudflare/cfssl
#
#     gpg: sudo apt install gnupg1 (IMPORTANT: Must be version 1. May need to symlink 'gpg' to 'gpg1' binary once installed)
#     ln -s /usr/bin/gpg1 /usr/local/bin/gpg
#
#   3. Ensure Hypershift tugboat KUBECONFIG file is available in the <HYPERSHIFT_ROOT> folder
#     export GOPATH=/performance
#     cd $HYPERSHIFT_ROOT
#     /performance/bin/armada-perf-client2 cluster config --cluster <hypershift-management-cluster> --admin
#     Extract the zip file into the $HYPERSHIFT_ROOT folder and create symbolic link to "openshift-test-management-cluster-kubeconfig"
#       unzip -j kubeConfigAdmin-<hypershift-management-cluster>.zip
#       ln -s kube-config-dal09-<hypershift-management-cluster>.yml openshift-test-management-cluster-kubeconfig
#
# One time hypershift setup
# -------------------------
#
#   Steps 1 to 3 should be performed once for each thread required.
#
#   1. Clone required github repos (will need to setup ssh keys for access)
#     cd $HYPERSHIFT_ROOT (See variables section below)
#     mkdir repo# (where # is the thread number); e.g. mkdir repo1
#     cd repo#
#     git clone --recurse-submodules git@github.ibm.com:alchemy-containers/armada-openshift-master.git -b <hypershift-branch>
#     (Check what branch to use with Tyler - e.g. hypershift-named-cert-update-newapi)
#
#     git clone git@github.ibm.com:alchemy-containers/armada-ansible.git
#
#   2. Create python virtual environment
#   cd $HYPERSHIFT_ROOT/repo#/armada-openshift-master/
#   python3.8 -m venv .venv --prompt "py3-$(basename $(pwd))"
#   source .venv/bin/activate
#   pip install --upgrade pip setuptools wheel
#
#   3. Build dependencies
#     make python-deps
#     make galaxy-deps
#

# Parameters
# ----------
# 1. Cluster Prefix - Prefix to be used for the name of the created Openshift clusters - Required
# 2. Tugboat (management cluster) type (iks or openshift)
# 3. BOM Version - Openshift version (e.g. 4.8) - Required
# 4. Count - Total number of Openshift clusters to create - Optional. Default: 1
# 5. Threads - Number of parallel threads to use for the creation of the Openshift clusters - Optional. Default: 1

CLUSTER_PREFIX=$1
TUGBOAT_TYPE=$2
BOM_VERSION=$3
COUNT=$4
THREADS=$5

if [[ -z ${CLUSTER_PREFIX} ]]; then
    echo "ERROR: Please specify a hypershift hosted cluster name prefix"
    exit 1
fi

if [[ -z ${TUGBOAT_TYPE} ]]; then
    echo "ERROR: Please specify the type of tugboat (management cluster) 'iks' or 'openshift'"
    exit 1
else
    TUGBOAT_TYPE=${TUGBOAT_TYPE,,} # ensure lower case

    if [[ ${TUGBOAT_TYPE} != "iks" && ${TUGBOAT_TYPE} != "openshift" ]]; then
        echo "ERROR: Tugboat type should be 'iks' or 'openshift'"
        exit 1
    fi
fi

if [[ -z ${BOM_VERSION} ]]; then
    echo "ERROR: Please specify an Openshift version for the hosted clusters (e.g. 4.8)"
    exit 1
fi

if [[ -z ${HYPERSHIFT_ROOT} ]]; then
    export HYPERSHIFT_ROOT=$HOME/hypershift
fi

if [[ -z ${WORKSPACE} ]]; then
    WORKSPACE=/performance/armada-perf
fi

if [[ -z ${COUNT} ]]; then
    COUNT=1
fi

if [[ -z ${THREADS} ]]; then
    THREADS=1
fi

CLUSTERS_PER_THREAD=${COUNT}
if [[ ${THREADS} -gt ${COUNT} ]]; then
    THREADS=${COUNT}
fi

printf "%s - Creating %s Openshift %s hosted cluster(s) using %s thread(s)\n" "$(date +%FT%T)" "${COUNT}" "${BOM_VERSION}" "${THREADS}"

if [[ ${THREADS} -gt 1 ]]; then
    CLUSTERS_PER_THREAD=$((COUNT / THREADS))
    PLUS_ONE_THREADS=$((COUNT - CLUSTERS_PER_THREAD * THREADS))
fi

PLUS_ONE=1
for ((THREAD = 1; THREAD <= ${THREADS}; THREAD = THREAD + 1)); do
    echo "Initiating creation of ${CLUSTERS_PER_THREAD} hypershift clusters in thread ${THREAD}."
    if [[ ${THREAD} -gt ${PLUS_ONE_THREADS} ]]; then
        PLUS_ONE=0
    fi

    ${WORKSPACE}/tools/hypershift_cluster_thread.sh ${THREAD} ${CLUSTER_PREFIX} ${TUGBOAT_TYPE} ${BOM_VERSION} $((CLUSTERS_PER_THREAD + PLUS_ONE)) >hypershiftClusterCreate-${CLUSTER_PREFIX}-${THREAD}.txt 2>&1 &

    sleep 120
done

jobs
echo
printf "%s - Waiting on %s threads to complete\n" "$(date +%FT%T)" "${THREADS}"
wait

printf "%s - Complete\n" "$(date +%FT%T)"
