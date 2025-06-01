#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Change the machine type for all workers in a pool

zone=dal09
workers=3
# poolName must be either firewall, edge or compute
poolName=firewall
clusterName=privCluster3
privateVlan=2263901
publicVlan=2263903

firewallLabels="dedicated=firewall,node-role.kubernetes.io/firewall=true,ibm-cloud.kubernetes.io/private-cluster-role=gateway"
workerLabels="node-role.kubernetes.io/compute=true,ibm-cloud.kubernetes.io/private-cluster-role=worker"
edgeLabels="dedicated=edge,node-role.kubernetes.io/edge=true,ibm-cloud.kubernetes.io/private-cluster-role=worker"

case ${poolName} in
"firewall") labels="${firewallLabels}";;
"edge") labels="${edgeLabels}";;
"compute") labels="${computeLabels}";;
*)
  echo "ERROR: poolName ${poolName} isn't either firewall, edge or compute"
  exit 1
  ;;
esac

if [[ $# -lt 1 ]]; then
    echo "ERROR: Must specify machine type"
    exit 1
fi

machineType=$(ibmcloud ks flavors --zone ${zone} -s | grep $1 | awk '{print $1}')

if [[ -z ${machineType} || ${machineType} != $1 ]]; then
    echo "ERROR: Matchine type $1 not available in zone ${zone}"
    exit 1
fi

ibmcloud ks worker-pool get -s --cluster ${clusterName} --worker-pool ${poolName} 1> /dev/null 2>&1
if [[ $? -eq 0 ]]; then
    ibmcloud ks worker-pool get -s --cluster ${clusterName} --worker-pool ${poolName} | grep "Machine Type" | egrep "${machineType}($|.encrypted)" > /dev/null 2>&1
    if [[ $? -eq 0 ]]; then
        echo "${poolName} worker pool is already setup for machine type ${machineType}"
        exit 0
    fi
    echo "Deleting ${poolName} worker pool"
    ibmcloud ks worker-pool rm -s --cluster ${clusterName} --worker-pool ${poolName}
    echo "Waiting for ${poolName} worker pool to disapear ...."
    while (true); do
        ibmcloud ks worker-pool get -s --cluster ${clusterName} --worker-pool ${poolName} 2>&1 | grep "The specified worker pool could not be found" 1> /dev/null 2>&1
        if [[ $? -eq 0 ]]; then
            break
        fi
        sleep 20
    done
fi

echo "Create ${poolName} worker pool with ${workers} ${machineType} workers"
ibmcloud ks worker-pool create classic -s --cluster ${clusterName} --name ${poolName} --machine-type ${machineType} --size-per-zone ${workers} --hardware shared --labels ${labels}

sleep 20

echo "Adding worker pool to ${zone} zone"
ibmcloud ks zone add classic -s --cluster ${clusterName} -p ${poolName} --zone ${zone} --private-vlan ${privateVlan} --public-vlan ${publicVlan}

echo "Execute something close to the following command, check machine types and counts before running command"
echo "./create_private_cluster.sh --name ${clusterName} --region us-south --firewall-zones ${zone} --compute-zones ${zone} --edge-zones ${zone} --public-vlan ${publicVlan} --private-vlan ${privateVlan} -fm ${machineType} -cm c2c.16x16 -em c2c.16x16 -f 3 -c 18 -e 3"
