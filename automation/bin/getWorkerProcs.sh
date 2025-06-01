#!/bin/bash -e

# KUBECONFIG environment must be set

if [[ -z "${KUBECONFIG}" ]]; then
    printf "KUBECONFIG not set. Exiting.\n"
    exit 1
fi

OIFS=$IFS
IFS=$'\n'

declare -i nPlatinum=0
declare -i nGold=0
declare -i nSlow=0

echo "Worker node processors for cluster:"
pods=$(kubectl get pods -A -o=wide | grep "calico-node" | sort -h -k8)
for pod in $pods; do
    namespace=$(echo ${pod} | awk '{print $1}')
    podName=$(echo ${pod} | awk '{print $2}')
    node=$(echo ${pod} | awk '{print $8}')
    set +e
    processor=$(kubectl exec --request-timeout 60s -n ${namespace} ${podName} -- cat /proc/cpuinfo | grep -m1 "model name")
    set -e
    echo "Node: ${node}, ${processor}"
    if [[ ${processor} == *"Platinum"* ]]; then
        nPlatinum=${nPlatinum}+1
    elif [[ ${processor} == *"Gold"* ]]; then
        nGold=${nGold}+1
    else
        nSlow=${nSlow}+1
    fi
done

echo "Total number of Platinum Processors: ${nPlatinum}"
echo "Total number of Gold Processors: ${nGold}"
echo "Total number of slow Processors: ${nSlow}"

IFS=${OIFS}
