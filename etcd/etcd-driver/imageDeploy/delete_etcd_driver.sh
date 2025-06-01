#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018, 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to delete etcd-driver deployments
# Prereqs - Kubectl with an appropriate KUBECONFIG set, and helm must be already
#  configured.

USAGE="Usage: `basename $0` <etcd-driver name prefix> <namespace>|all [<helm chart, defaults to 'etcd-driver'>]"

if [[ $# -lt 2 ]]; then
    echo $USAGE
    exit
fi

PREFIX=$1
if [[ $2 == "all" ]]; then
  NAMESPACE=""
else
  NAMESPACE="--namespace $2"
fi

helm_chart=etcd-driver
if [[ $# -ge 3 ]]; then
    helm_chart=$3
fi

echo "Going to delete all etcd-driver clusters with prefix $PREFIX in namespace $2"

# Delete any etcd-driver deployments
for c in `helm list $NAMESPACE | grep $PREFIX | grep ${helm_chart} | awk '{print $1}' `; do
    echo "Deleting etcd-driver $c in namespace $NAMESPACE"
    helm uninstall $c ${NAMESPACE}
done
