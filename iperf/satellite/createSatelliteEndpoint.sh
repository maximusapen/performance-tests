#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# Create location or cloud endpoints for iperfserver

# You need to have logged into IKS before running this script

if [[ $# -lt 3 ]]; then
    echo
    echo "Usage: ./createSatelliteEndpoint.sh <endpointPrefix> <cloud | location> <number_of_service> "
    echo
    exit 1
fi

# endpointPrefix is the name you choose for the endpoint name like name=${endpointPrefix}-$id-${endpointType}-endpoint
declare endpointPrefix=$1
# Endpoint Type can be location or cloud
declare endpointType=$2
# Number of endpoints to create
declare -i maxService=$3

# Set up location id in location_id_file
source envFile

if [[ -z ${SERVER_KUBECONFIG} ]]; then
    echo "You need to export SERVER_KUBECONFIG"
    exit 1
fi

echo Using SERVER_KUBECONFIG ${SERVER_KUBECONFIG}
export KUBECONFIG=${SERVER_KUBECONFIG}

dest_protocol=TCP
source_protocol=TCP

# Save all pods and service in default namespace to tmp files
kubectl get pod -o wide | grep server >/tmp/iperfServers
kubectl get service | grep NodePort >/tmp/iperfServices
echo "IPerf servers found:"
cat /tmp/iperfServers
echo "IPerf services found:"
cat /tmp/iperfServices

for i in $(seq 1 ${maxService}); do
    # Get deployment and service
    iperfServer=$(grep iperfserver-$i-deployment /tmp/iperfServers)
    iperfService=$(grep "iperfserver-np-service-$i " /tmp/iperfServices)
    # Get external host
    host=$(echo $iperfServer | awk '{print $7}')
    dest_host=$(kubectl get node -o wide | grep ${host} | awk '{print $7}')
    echo host: $host dest_host $dest_host
    dest_port=$(echo $iperfService | awk '{print $5}' | sed "s/5201://" | sed "s/\/TCP//") # for iperf3
    #dest_port=$(echo $iperfService | awk '{print $5}' | sed "s/5001://" | sed "s/\/TCP//")  # for iperf2 using iperf2 images
    echo
    echo "iperfServer: $iperfServer"
    echo "iperfService: $iperfService"
    name=${endpointPrefix}-$i-${endpointType}-endpoint
    echo Creating satellite endpoint ${name} in location ${location_id} for $dest_host:$dest_port
    ibmcloud sat endpoint create --location ${location_id} --name ${name} --dest-type ${endpointType} --dest-hostname ${dest_host} --dest-port ${dest_port} --dest-protocol ${dest_protocol} --source-protocol ${source_protocol}
done

echo Checking endpoints created with command
echo ibmcloud sat endpoint ls --location ${location_id}
echo grep ${endpointPrefix} to get the endpoints created
ibmcloud sat endpoint ls --location ${location_id} | grep ${endpointPrefix}
echo
echo Check endpoints with the following command until hostname and ports are created before use
echo
echo ibmcloud sat endpoint ls --location ${location_id}
echo
