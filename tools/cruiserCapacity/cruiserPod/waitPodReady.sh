#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Set up your KUBECONFIG before running this script
#export KUBECONFIG=<cruiser kube config file>

while [ 1 ]; do
    nr=$(kubectl get pod --all-namespaces | grep -v Running | grep -v NAME | wc -l)
    echo Waiting for $nr pods Running
    if [[ $nr = "0" ]]; then
        break
    fi
    sleep 60
done

