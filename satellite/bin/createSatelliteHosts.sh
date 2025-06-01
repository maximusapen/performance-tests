#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020, 2023 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# Script to create Satellite hosts
if [[ -z "$1" ]]; then
	echo "Please provide a location name as the first parameter"
  exit 1
else
	echo "Creating VSIs for location $1"
	LOCATIONNAME=$1
fi

DOMAIN=satellite
# Note that on macs this wont work if you don't have gnu sed install gsed
SED_COMMAND=gsed

# you can create and get this ssh key id using the ibmcloud sl security sshkey-add/sshkey-list
HOST_SSH_KEY_ID="2176464"

CONTROL_HOSTS_PER_ZONE=10
CONTROL_CPU=16
CONTROL_MEMORY=65536
CONTROL_DISK=100
CONTROL_NETWORK=1000

WORKER_HOSTS_PER_ZONE=1
WORKER_CPU=4
WORKER_MEMORY=16384
WORKER_DISK=100
WORKER_NETWORK=1000

# Account login config
PROD_APIKEY=${PROD_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}
PROD_ACCOUNT=${armada_performance_prod_account_id}

ibmcloud_login_prod() {
# Replace with whatever login mechanism works for your account
  ibmcloud login -a https://cloud.ibm.com --apikey $PROD_APIKEY  -c $PROD_ACCOUNT -r us-south
}

create_vm () {
    ibmcloud sl vs create -H $1 -D $2 --os REDHAT_7_64 -c $3 -m $4 --datacenter $5 --vlan-public $6 --vlan-private $7 --disk $8 --network $9 -f --key $HOST_SSH_KEY_ID;   
}

ibmcloud_login_prod

# Iterate over datacenters
jq -c '.[]' datacenters.json | while read dc; do
    HOST_DATACENTER=$(echo $dc | jq ' .dc' | tr -d '"')
    HOST_PUBLIC_VLAN=$(echo $dc | jq ' .public_vlan' | tr -d '"')
    HOST_PRIVATE_VLAN=$(echo $dc | jq  '.private_vlan' | tr -d '"')

    # Create control plane hosts
    for ((i=1; i<=${CONTROL_HOSTS_PER_ZONE}; i++))
    do
      create_vm "arm-${LOCATIONNAME}-${HOST_DATACENTER}-cont-$i" ${LOCATIONNAME}.${DOMAIN} ${CONTROL_CPU} ${CONTROL_MEMORY} ${HOST_DATACENTER} ${HOST_PUBLIC_VLAN} ${HOST_PRIVATE_VLAN} ${CONTROL_DISK} ${CONTROL_NETWORK}
    done

    # Create Worker plane hosts
    for ((i=1; i<=${WORKER_HOSTS_PER_ZONE}; i++))
    do
      create_vm "arm-${LOCATIONNAME}-${HOST_DATACENTER}-work-$i" ${LOCATIONNAME}.${DOMAIN} ${WORKER_CPU} ${WORKER_MEMORY} ${HOST_DATACENTER} ${HOST_PUBLIC_VLAN} ${HOST_PRIVATE_VLAN} ${WORKER_DISK} ${WORKER_NETWORK}
    done
done
