#!/bin/bash


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
