#! /bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
readonly API_SERVER_IP=127.0.0.1
readonly API_SERVER_PORT=6969

readonly API_VERSION=v1

readonly ARMADA_URL=http://${API_SERVER_IP}:${API_SERVER_PORT}/${API_VERSION}/clusters

readonly DUMMY_UAA_TOKEN="bearer "$(<dummy_uaa_token.jwt)

totalClusters=${1:-1}

# Base identifier from which we'll create a unique org id for each cluster.
orgBase=a1b2cdef-6a7f-1569-9f29-000000000000

for ((clusterNum=1;clusterNum<=totalClusters;clusterNum++)); do
  clusterNumLen=${#clusterNum}
  clusterName=perfCluster${clusterNum}

  orgId=${orgBase::${#orgBase}-${clusterNumLen}}${clusterNum}

  # Generate http request
  curlCmd="curl -H \"X-Auth-UAA-Token: ${DUMMY_UAA_TOKEN}\" -H \"X-Auth-Resource-Id: ${orgId}\" -X GET ${ARMADA_URL}"
  echo ${curlCmd}
  eval ${curlCmd}
done
