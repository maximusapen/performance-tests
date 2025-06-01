#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2018, 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to deploy etcd-driver instances using helm
# Prereqs - Kubectl with an appropriate KUBECONFIG set, and helm must already be configured.

if [[ $# -lt 2 ]]; then
    echo "Usage: `basename $0` <prefix> <namespace> [<helm chart name, defaults to 'etcd-driver'>]"
    echo "<prefix> = The prefix to match etcd-operator instances. etcd-driver, or <helm chart name>, instances will be created for every etcd-operator that matches the prefix (using grep)"
    echo "<namespace> = The namespace to create etcd-drivers in"
    echo "[<helm chart>] = The helm chart (defaults to etcd-driver)"
    exit 1
fi

# Only needed for 'VIP_DNS#'
. etcd-perftest-config

prefix=$1
namespace=$2

helm_chart=etcd-driver
if [[ $# -eq 3 ]]; then
    helm_chart=$3
fi

dt=$(date +"%Y-%m-%d-%H-%M")

#If this scripts were to run just against a single etcdcluster then this would be the way to set the endpoints
#endpoints="--set parameters.endpoints=${ETCD_VIP_ENDPOINTS//,/\\,}"

pods="--set parameters.pods=50"
connections="--set parameters.clients=20 --set parameters.conns=20"
pattern="--set parameters.pattern=/prefix/:ip/%level2-%06d[2]/%level3-%04d[6]/%level4-%04d[2]/%level5-%040d[100]/%level6-%040d[10]/%leaf7-%06d[5];[0-9]{10\,30} --set parameters.valSpec=10\,30"
# To disable 1 container or the other:--set parameters.watchLevelCounts="" and/or --set parameters.churnContainer=""
churn="--set parameters.churnValRate=7920 --set parameters.churnLevelRate=250 --set parameters.churnLevel=5 --set parameters.churnLevelPct=5 --set parameters.getLevel=7"
watch="--set parameters.watchPrefixGetInterval=30s --set parameters.putRate=244316"
gets="--set parameters.getRate=450000"
leases="--set parameters.leaseStartupDelay=33m"
# Setting parameters.runId causes csv results to be put into /results/${dt}/[watch|deploy]/ directory in the pvc
#others="--set parameters.runId=${dt}"

echo "Start ${dt} - Also runId"
for c in `kubectl get etcdclusters -n ${namespace} --no-headers | grep ${prefix} | awk '{print $1}' `; do
    jobname=${c}-etcd-drv
    echo "Creating ${helm_chart} ${jobname} to run against etcdcluster $c in namespace ${namespace}"
    # Cleanup any old deployments
    helm uninstall ${jobname} --namespace ${namespace}

      
    kubectl get secrets -n ${namespace} --no-headers | grep ${c}-client-tls > /dev/null
    if [[ $? -eq 0 ]]; then
        echo "Using secrets"
        secretsPrefix="--set secretsPrefix=${c}"
    fi

    node_port=$(kubectl -n armada get svc ${c}-client-service-np -o=jsonpath='{.spec.ports[*].nodePort}{"\n"}')
    endpoints="--set parameters.endpoints=${VIP_DNS1}:${node_port}\,${VIP_DNS2}:${node_port}\,${VIP_DNS3}:${node_port}"

    # Use helm to install the chart and run driver
    helm install ${jobname} ${helm_chart} --namespace=${namespace} --set prefix=${c} ${secretsPrefix} --set namespace=${namespace} ${endpoints} ${pods} ${connections} ${pattern} ${churn} ${watch} ${gets} ${others}
done
sleep 5
