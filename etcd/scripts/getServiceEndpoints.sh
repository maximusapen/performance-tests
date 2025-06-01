#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

port=`kubectl get services | grep etcd-slnfs-public | awk '{print $4}' | cut -d"/" -f1 | cut -d":" -f2`
SERVICE_ENDPOINTS=""
for node in `kubectl get nodes | cut -d" " -f1`; do
         if [ "$node" != "NAME" ]
         then
             if [[ -z $SERVICE_ENDPOINTS ]]; then
                SERVICE_ENDPOINTS="${node}:${port}"
             else
                SERVICE_ENDPOINTS="${SERVICE_ENDPOINTS},${node}:${port}"
             fi
         fi
done
echo "$SERVICE_ENDPOINTS"
