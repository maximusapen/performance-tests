#!/bin/bash -ex
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Set up your KUBECONFIG before running this script
#export KUBECONFIG=<cruiser kube config file>

pods=$(kubectl get pods | grep getnodeaccess | awk '{print $1}')

for pod in ${pods[@]}
do
    echo Deleting $pod
    kubectl delete pod $pod
done

