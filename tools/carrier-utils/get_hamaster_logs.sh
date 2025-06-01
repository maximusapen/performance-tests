#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to gather logs from all pods for an ha master
# Prereqs - Kubectl with an appropriate KUBECONFIG set

if [[ $# -ne 2 ]]; then
    echo "Usage: `basename $0` <prefix> <master Pod Containers>"
    echo "<prefix> = The prefix to match ha master pods with"
    echo "<master Pod Containers> = the containers in the master pod to gather logs from - specify either all to get all containers or a comma seperated list."
    exit 1
fi

prefix=$1
DATE=$(date -u +%FT%TZ)

containers=$2
if [[ $containers == "all" ]]; then
  containersArray=(apiserver scheduler controller-manager kube-addon-manager vpn)
else
  OIFS=$IFS
  IFS=',' read -r -a containersArray <<< "$containers"
  IFS=$OIFS
fi

OIFS=$IFS
IFS=$'\n'

loggedOperators=()

pods=$(kubectl get pods --all-namespaces --no-headers -o=wide | grep ${prefix})
echo "$pods" > $prefix.podInfo.$DATE.log
for pod in $pods; do
    namespace=$(echo ${pod}| awk '{print $1}')
    podName=$(echo ${pod}| awk '{print $2}')
    echo "Found pod $podName in namespace $namespace"
    if [[ $namespace == kubx-etcd* ]]; then
        kubectl logs -n $namespace $podName > $podName-$namespace.$DATE.log
        if [[ ! " ${loggedOperators[@]} " =~ " ${namespace} " ]]; then
            # Only log etcd-operator pod once
            etcdOperPod=$(kubectl get pods -n $namespace | grep etcd-operator | awk '{print $1}')
            echo "Dumping logs for $etcdOperPod in $namespace"
            kubectl logs -n $namespace $etcdOperPod > $etcdOperPod-$namespace.$DATE.log
            loggedOperators+=($namespace)
        fi
    elif [[ $namespace == "kubx-masters" ]]; then
        if [[ $podName == openvpnserver-* ]]; then
            kubectl logs -n $namespace $podName > $podName-openvpn.$DATE.log
        else
          for element in "${containersArray[@]}"
          do
              kubectl logs -n $namespace $podName -c $element> $podName-$element.$DATE.log
          done

        fi
    else
        echo "Pod $podName in namespace $namespace will be ignored"
    fi
done

IFS=$OIFS
