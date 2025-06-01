#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# Run one iperfclient with specified id and satellite endpoint for duration in seconds

# Set up client and sever kubeconfig as follows before running this script
# export CLIENT_KUBECONFIG=<full path to file>/kube-config-<location>-<<name of your client cluster>.yml

if [[ $# -lt 4 ]]; then
    echo
    echo "Usage: ./runClientEndpoint.sh <endpointPrefix> <cloud | location> <client_id> <duration_in_seconds> "
    echo
    exit 1
fi

# Specified the endpointPrefix used in createtSatelliteEndpoint.sh
endpointPrefix=$1
# Endpoint Type can be location or cloud
endpointType=$2
# ID of iperfclient to run
id=$3
# Duration to run in seconds
duration=$4

# Set up env in envFile
source envFile

if [[ -z ${CLIENT_KUBECONFIG} ]]; then
    echo "You need to export CLIENT_KUBECONFIG"
    exit 1
fi

echo Using CLIENT_KUBECONFIG ${CLIENT_KUBECONFIG}
export KUBECONFIG=${CLIENT_KUBECONFIG}

ibmcloud sat endpoint ls --location ${location_id} | grep ${endpointPrefix} | grep ${endpointType} >/tmp/satendpoints
if [[ $? -ne 0 ]]; then
    echo "Command failed: ibmcloud sat endpoint ls --location ${location_id}"
    echo "You may have to run ibmcloud login again.  Then if this is stage environment, configure to use:"
    echo "    ibmcloud ks init --host origin.containers.test.cloud.ibm.com"
    exit 1
fi

echo "Number of endpoints found:"
cat /tmp/satendpoints

date

echo Processing client $id: ${endpointPrefix}-$id
endpoint=$(grep "${endpointPrefix}-$id" /tmp/satendpoints)
address=$(echo $endpoint | awk '{print $5}' | sed "s/:/ /")
host=$(echo $address | awk '{print $1}')
port=$(echo $address | awk '{print $2}')
echo iperfclient.sh --id $id --address $host -p $port -t $duration ${iperf_args} -J
${perf_dir}/iperf/bin/iperfclient.sh --id $id --address $host -p $port -t $duration ${iperf_args} -J
