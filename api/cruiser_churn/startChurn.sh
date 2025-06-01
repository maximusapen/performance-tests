#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2023 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to start cruiser churn. It has 1 optional parameter, which is 'mode'.
# mode should be 'real', 'fake', `realopenshift` or `fakeopenshift`.
if [[ -z "$1" ]]; then
    # Default is fake cruisers (0 workers)
    mode="fake"
else
    mode=$1
fi


PERF_DIR=/performance
CHURN_SRC_DIR=${PERF_DIR}/armada-perf/api/cruiser_churn
CHURN_DIR=${PERF_DIR}/stats/churn
mkdir -p ${CHURN_DIR}
#Calculate carrier from perf client we are running on
CLI=$(hostname | cut -d "-" -f3)
C_NUM="${CLI: -1}"
if [[ ${C_NUM} -eq 1 ]]; then
    C_NUM=5
fi
ENV=$(hostname | cut -d "-" -f1)

export KUBECONFIG=/performance/config/carrier${C_NUM}_${ENV}/admin-kubeconfig
export GOPATH=${PERF_DIR}
OVERRIDE_ENV_FILE=$CHURN_DIR/carrier${C_NUM}.env
ENV_FILE=${CHURN_SRC_DIR}/carrier${C_NUM}.env
CARRIER_NAME="carrier${C_NUM}_stage"

if [ -f "${OVERRIDE_ENV_FILE}" ]; then
    echo "Using ${OVERRIDE_ENV_FILE} for churn settings"
    source ${OVERRIDE_ENV_FILE}
elif [ -f "${ENV_FILE}" ]; then
    echo "Using ${ENV_FILE} for churn settings"
    source ${ENV_FILE}
else
    echo "Unable to find file ${OVERRIDE_ENV_FILE} or ${ENV_FILE}, exiting"
    exit 1
fi

LOG_FILE=cruiser_churn.log

if [[ ${mode} == "fake" ]]; then
    THREADS=${FAKE_THREADS}
    CLUSTERS=${FAKE_CLUSTERS}
    WORKERS="-1"
    MACHINETYPE="u3c.2x4"
    ARGS_STRING="-followKubeVersion"
    PREFIX="fakecruiser-churn-"
    TESTNAME="cruiserchurn"
elif [[ ${mode} == "real" ]]; then
    THREADS=${REAL_THREADS}
    CLUSTERS=${REAL_CLUSTERS}
    WORKERS="1"
    MACHINETYPE="u3c.2x4"
    ARGS_STRING="-workerPollInterval 30s -followKubeVersion"
    PREFIX="realcruiser-churn-"
    TESTNAME="cruiserchurn"
elif [[ ${mode} == "fakeopenshift" ]]; then
    THREADS=${FAKE_OPENSHIFT_THREADS}
    CLUSTERS=${FAKE_OPENSHIFT_CLUSTERS}
    WORKERS="-1"
    MACHINETYPE="b3c.4x16"
    ARGS_STRING="-defaultKubeVersion 4.10_openshift -upgradeKubeVersion 4.11_openshift"
    PREFIX="fakeopenshift4-churn-"
    TESTNAME="cruiserchurn_openshift"
    LOG_FILE=cruiser_churn.tugboat.log
elif [[ ${mode} == "realopenshift" ]]; then
    THREADS=${REAL_OPENSHIFT_THREADS}
    CLUSTERS=${REAL_OPENSHIFT_CLUSTERS}
    WORKERS="1"
    MACHINETYPE="b3c.4x16"
    ARGS_STRING="-workerPollInterval 30s -defaultKubeVersion 4.10_openshift"
    PREFIX="realopenshift4-churn-"
    TESTNAME="cruiserchurn_openshift"
    LOG_FILE=cruiser_churn.tugboat.log
else

    echo "First parameter should be mode - use either real, fake, realopenshift or fakeopenshift"
    exit 1
fi

echo "Starting churn on carrier${C_NUM} with ${CLUSTERS} ${mode} cruisers & ${THREADS} threads"
${PERF_DIR}/bin/cruiser_churn -action ChurnClusters -carrierName ${CARRIER_NAME} -clusterNamePrefix ${PREFIX} -deleteResources -testname ${TESTNAME} -clusters ${CLUSTERS} -workers ${WORKERS} -machineType ${MACHINETYPE} -numThreads ${THREADS} -monitor -metrics -verbose=false -masterPollInterval 30s ${ARGS_STRING} >> ${CHURN_DIR}/${LOG_FILE} 2>&1
