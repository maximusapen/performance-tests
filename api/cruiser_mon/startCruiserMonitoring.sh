#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Run to start cruiser_mon
# Takes 1 mandatory parameter, which should be 'carrier', 'tugboat' or 'satellite' to monitor carrier, tugboat cruisers or satellites
# For monitoring Satellite clusters, takes an additional optional parameter to specify the name of the Satellite location (used for metrics naming)
if [[ -z "$1" ]]; then
    echo "Specify the type of cluster to monitor (carrier, tugboat or satellite"
    exit 1
else
    if [[ $1 == "tugboat" ]]; then
        target="tugboat"
    elif [[ $1 == "satellite" ]]; then
        target="satellite"
    elif [[ $1 == "hypershift" ]]; then
	    target="hypershift"
    else
        target="carrier"
    fi
fi

if [[ -n "$2" ]]; then
    export METRICS_LOCATION="$2"
fi

echo "Cruiser_mon is monitoring cruisers on a $target"

PERF_DIR=/performance
CMON_DIR=${PERF_DIR}/stats/cruiser_mon
CONFIG_DIR=${CMON_DIR}/carrier-nfs
sudo rm -rf ${CONFIG_DIR}
sudo mkdir -p ${CONFIG_DIR}
sudo chmod 777 ${CONFIG_DIR}
mkdir -p ${CMON_DIR}
#Calculate carrier from perf client we are running on
CLI=$(hostname | cut -d "-" -f3)
C_NUM="${CLI: -1}"
ENV=$(hostname | cut -d "-" -f1)
LOG_FILE=cruiser_mon.log

if [ "$target" != "carrier" ]; then
    C_NUM="${C_NUM}00"
    # Note - if specifying a different measurement name - always ensure it starts with dummycruisermaster,
    # otherwise metrics code will not retry the influxdb connection each time if it fails.
    MEASUREMENT_STRING="--measurement dummycruisermaster_${target}"
    LOG_FILE=cruiser_mon.${target}.log
fi

if [ "$target" == "satellite" ]; then
    export KUBECONFIG=/performance/config/satellite0/admin-kubeconfig
elif [ "$target" == "hypershift" ]; then
    # Hacky - this needs to point to the Kubeconfig for the hypershift management cluster
    export KUBECONFIG=/performance/config/rgs-large-iks-48-1/kube-config-dal09-rgs-large-iks-48-1.yml
else
    export KUBECONFIG=/performance/config/carrier${C_NUM}_${ENV}/admin-kubeconfig
fi
export GOPATH=${PERF_DIR}
${PERF_DIR}/bin/cruiser_mon -prefix '' --timeout 30s --loop 60s -dir ${CONFIG_DIR} ${MEASUREMENT_STRING} >>${CMON_DIR}/${LOG_FILE} 2>&1
