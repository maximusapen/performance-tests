#!/bin/bash

# This script will enable logDNA for the cluster set by KUBECONFIG

# Usage:  enableLogDNA.sh [Openshift]
# Example:  For default clusters:
#           - enableLogDNA.sh
# Example:  For openshift clusters:
#           - enableLogDNA.sh Openshift
clusterPlatform=$1
echo clusterPlatform: ${clusterPlatform}

if [[ ${clusterPlatform} == "Openshift" ]]; then
    namespace="ibm-observe"
else
    namespace="default"
fi

# KUBECONFIG environment must be set
if [[ -z "${KUBECONFIG}" ]]; then
    printf "KUBECONFIG not set. Exiting.\n"
    exit 1
fi

clusterName=$(kubectl config current-context)

# Create LogDNA pods for default clusters
createLogDNAPods() {
    echo "Creating LogDNA Pods for default clusters"
    kubectl create secret generic logdna-agent-key --from-literal=logdna-agent-key=${armada_performance_logdna_ingestion_key}
    kubectl create -f https://assets.us-south.logging.cloud.ibm.com/clients/logdna-agent-ds.yaml

}

# Create LogDNA pods for Openshift clusters
createLogDNAPodsOpenshift() {
    echo "Creating LogDNA Pods for Openshift clusters"
    armada_perf_dir=/performance/armada-perf
    oc adm new-project --node-selector='' ibm-observe
    oc create serviceaccount logdna-agent -n ibm-observe
    oc adm policy add-scc-to-user privileged system:serviceaccount:ibm-observe:logdna-agent
    oc create secret generic logdna-agent-key --from-literal=logdna-agent-key=${armada_performance_logdna_ingestion_key} -n ${namespace}
    oc create -f ${armada_perf_dir}/automation/bin/logdna-agent-ds-os.yaml -n ibm-observe
}

# Redirect stderr to /dev/null for "No Resources found." when there are no pods in namespace
nAgents=$(kubectl get pods -n ${namespace} 2>/dev/null | grep logdna-agent | wc -l | awk '{print $1}')
if [[ ${nAgents} == "0" ]]; then
    echo "Enabling logDNA for ${clusterName}"

    # Ignore enable errors for now.
    # Check for agent pods below will show whether logDNA is enabled.
    createLogDNAPods${clusterPlatform}

    # Openshift cluster may take longer to create logdna-agent pods
    sleep 30
    nAgents=$(kubectl get pods -n ${namespace} | grep logdna-agent | wc -l | awk '{print $1}')
fi

echo "There are ${nAgents} logdna-agent"
# logdna-agent pods are in default namespace
kubectl get pods -n ${namespace} | grep logdna-agent

if [[ ${nAgents} == "0" ]]; then
    echo
    echo "WARNING WARNING WARNING WARNING WARNING WARNING WARNING WARNING WARNING"
    echo "WARNING:   Failed to enable logDNA. No logdna-agent created.    WARNING"
    echo "WARNING:   Continue to run test as we are not testing logDNA.   WARNING"
    echo "WARNING WARNING WARNING WARNING WARNING WARNING WARNING WARNING WARNING"
    echo
    exit 0
fi

echo "logDNA is enabled for ${clusterName}"
echo
exit 0
