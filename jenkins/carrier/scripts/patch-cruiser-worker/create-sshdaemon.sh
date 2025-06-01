#!/bin/bash -ex
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

echo "Usage: . ./create-sshdaemon.sh <CLUSTER_NAME>"
CLUSTER_NAME=$1

echo "Removing old kubeConfig artefacts"
rm -fr kubeConfig*

echo "Getting cluster config"
${GOPATH}/bin/armada-perf-client -action=GetClusterConfig -clusterName=${CLUSTER_NAME} -admin
unzip kubeConfigAdmin-${CLUSTER_NAME}.zip

cd kubeConfig*
export KUBECONFIG=kube-config-dal09-${CLUSTER_NAME}.yml 
echo $KUBECONFIG

# deploy sshdaemon
kubectl apply -f ../sshdaemon.yaml

# get deployed sshdaemon
kubectl get pods -o wide

