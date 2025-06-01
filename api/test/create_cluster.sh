#! /bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
readonly API_SERVER_IP=127.0.0.1
readonly API_SERVER_PORT=6969

readonly API_VERSION=v1

readonly ARMADA_URL=http://${API_SERVER_IP}:${API_SERVER_PORT}/${API_VERSION}/clusters

readonly DUMMY_UAA_TOKEN="bearer "$(<dummy_uaa_token.jwt)
readonly DUMMY_SL_USERNAME="dummy_armadaemail@uk.ibm.com"
readonly DUMMY_SL_APIKEY="11111a11111111a111aa1111111a11111aa111a1aaaaaaa1a1111111aa1111aa" // pragma: allowlist secret

readonly POST_DATA_TEMPLATE=$(<cluster_create.json)

# Get number of clusters / workers per cluster from command line, defaulting to 1
totalClusters=${1:-1}
totalWorkers=${2:-1}

if [ $totalWorkers -gt 1 ]; then
  machineType="small"
else
  machineType="free"
fi
machineType=${3:-$machineType}

if [ $machineType != "free" ]; then
  softlayerCreds="-H \"X-Auth-Softlayer-Username: ${DUMMY_SL_USERNAME}\" -H \"X-Auth-Softlayer-APIKey: ${DUMMY_SL_APIKEY}\""
else
  softlayerCreds=""
fi

# Base identifier from which we'll create a unique org id for each cluster.
orgBase=a1b2cdef-6a7f-1569-9f29-000000000000
postDataSized="${POST_DATA_TEMPLATE/\%machineType\%/$machineType}"

for ((clusterNum = 1; clusterNum <= totalClusters; clusterNum++)); do
  clusterNumLen=${#clusterNum}
  clusterName=perfCluster${clusterNum}
  orgId=${orgBase::${#orgBase}-${clusterNumLen}}${clusterNum}

  # Generate http request from template, substituting the cluster name and worker count
  postData="${postDataSized/\%CLUSTERNAME\%/$clusterName}"
  postData="${postData/\"\%WORKERNUM\%\"/$totalWorkers}"

  curlCmd="curl -H \"X-Auth-UAA-Token: ${DUMMY_UAA_TOKEN}\" -H \"X-Auth-Resource-Id: ${orgId}\" ${softlayerCreds} -H \"Content-Type: application/json\" -X POST -d '${postData}' ${ARMADA_URL}"
  echo ${curlCmd}
  eval ${curlCmd}
done
