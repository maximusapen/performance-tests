#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to update a single master, and post the times to metrics service
# It assumes the environment has been correctly setup in advance.
# Generally this will be called from update_fake_cruisers.sh

if [[ $# -ne 2 ]]; then
    echo "Usage: `basename $0` <ID> <master_name> "
    exit 1
fi

cd ${WORKSPACE}/armada-ansible

ID=$1
MASTER_NAME=$2

echo "Updating cruiser $MASTER_NAME"
SECONDS=0
if [[ $THREADS -eq 1 ]]; then
    time -p ./deploy.py --env $REALENV --infra $CARRIER --managed ${MASTER_NAME} --updatekubX --targetbom ${BOM_VERSION} --verbose $EXTRA_DEPLOY_PARAMS --run-local
    STATUS=$?
    updateTime=$SECONDS
else
    time -p ./deploy.py --env $REALENV --infra $CARRIER --managed ${MASTER_NAME} --updatekubX --targetbom ${BOM_VERSION} --verbose $EXTRA_DEPLOY_PARAMS --run-local >> ${WORKSPACE}/fakeCruiserUpdateOut_${MASTER_NAME}.txt 2>&1
    STATUS=$?
    updateTime=$SECONDS
fi

# Deal with reality that metrics db has 30 second granularity.
if [[ $updateTime -lt 30 ]]; then
    sleep $((30-$updateTime))
fi
if [[ $STATUS -eq 0 ]]; then
    echo "UpdateTime: $updateTime"
    for (( j=1; j<=5; j++ )); do
        set +x
        curl -XPOST --header "X-Auth-User-Token: apikey $METRICS_KEY" -d "[{\"name\" : \"${DATACENTER}.performance.${CARRIER}_stage.DummyCruiserMaster.ID$ID.Update_Time.sparse-avg\",\"value\" : $updateTime}]" https://metrics.stage1.ng.bluemix.net/v1/metrics
        curl -XPOST --header "X-Auth-User-Token: apikey $METRICS_KEY" -d "[{\"name\" : \"${DATACENTER}.performance.${CARRIER}_stage.DummyCruiserMaster.ID$ID.Update_Success.count\",\"value\" : 1}]" https://metrics.stage1.ng.bluemix.net/v1/metrics
        set -x
        if [[ $? -eq 0 ]]; then
            echo "Posted cruiser update in $updateTime at `date +"%Y%m%d_%H%M%S"`"
            break
        fi
        echo "Metric post $j failed"
        sleep $j
    done
else
    echo "Update FAILED with return code of $STATUS in $updateTime seconds"
    echo "${MASTER_NAME}" >> ${WORKSPACE}/fakeCruiserUpdateFailures.txt
    for (( j=1; j<=5; j++ )); do
        set +x
        curl -XPOST --header "X-Auth-User-Token: apikey $METRICS_KEY" -d "[{\"name\" : \"${DATACENTER}.performance.${CARRIER}_stage.DummyCruiserMaster.ID$ID.Update_Failed.count\",\"value\" : 1}]" https://metrics.stage1.ng.bluemix.net/v1/metrics
        set -x
        if [[ $? -eq 0 ]]; then
            echo "Posted failed fake cruiser update `date +"%Y%m%d_%H%M%S"`"
            break
        fi
        echo "Metric post $j failed"
        sleep $j
    done
fi
