#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to scale cluster-updater instances

if [[ $# -ne 1 ]]; then
    echo "Usage: `basename $0` <prefix>"
    echo "<prefix> = The prefix to match for deployments to scale"
    exit 1
fi

prefix=$1

deployments=$(kubectl get deployments -n kubx-masters | grep "cluster-updater" | grep $prefix | awk '{print $1}')

OIFS=$IFS
IFS=$'\n'

for depl in $deployments; do
    echo "Scaling down $depl"
    kubectl scale deployment $depl -n kubx-masters --replicas=0
    sleep 2
done

echo "Sleeping for 1 minute"
sleep 60

for depl in $deployments; do
    echo "Scaling up $depl"
    kubectl scale deployment $depl -n kubx-masters --replicas=1
    sleep 2
done

IFS=$OIFS
