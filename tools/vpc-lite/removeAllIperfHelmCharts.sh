#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Helper script to remove all iperf namespace helm charts. Update cluster names before using.

NAMESPACE="iperf"

. privCluster
for c in `helm list --namespace $NAMESPACE | grep -v NAME | awk '{print $1}' `; do
    echo "Deleting  $c in namespace $NAMESPACE"
    helm uninstall $c --namespace $NAMESPACE
done

. iperfClient
for c in `helm list --namespace $NAMESPACE | grep -v NAME | awk '{print $1}' `; do
    echo "Deleting  $c in namespace $NAMESPACE"
    helm uninstall $c --namespace $NAMESPACE
done

. iperfServer
for c in `helm list --namespace $NAMESPACE | grep -v NAME | awk '{print $1}' `; do
    echo "Deleting  $c in namespace $NAMESPACE"
    helm uninstall $c --namespace $NAMESPACE
done
