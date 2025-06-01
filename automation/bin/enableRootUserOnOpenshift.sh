#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# This script will setup the required permissions for Openshift.
#
# Input Paramter:
# 1. Kubernetes namespace (required)
# 2. Bypass server certificate validation (optional - default false)

# KUBECONFIG environment must be set
if [[ -z "${KUBECONFIG}" ]]; then
    printf "KUBECONFIG not set. Exiting.\n"
    exit 1
fi

test_namespace=$1
if [[ -z "${test_namespace}" ]]; then
    printf "Kubernetes namespace not specified. Exiting.\n"
    exit 1
fi

oc_insecure_login=$2
if [[ -z "${oc_insecure_login}" ]]; then
    oc_insecure_login=false
fi

clusterName=$(kubectl config current-context)

set +e
kubectl get ns | grep openshift >/dev/null 2>&1
if [[ $? -eq 0 ]]; then
    set -e
    echo "Setting up ${test_namespace} namespace to support root users on openshift cluster"
    server=$(grep server: $KUBECONFIG | awk '{print $2}' | head -1)

    # Insure that oc is logged onto the correct server
    oc status 2>/dev/null | grep ${server}
    if [[ $? -eq 0 ]]; then
        oc project ${test_namespace}
        # Allow pods to run as any non-root ID
        oc adm policy add-scc-to-user nonroot -n ${test_namespace} -z default
        oc label --overwrite ns ${test_namespace} pod-security.kubernetes.io/warn=baseline

        if [ ${test_namespace} == persistent-storage ] || [ ${test_namespace} == k8s-netperf ] || [ ${test_namespace} == portworx-storage ] || [ ${test_namespace} == odf-storage ]; then
            # Persistent-storage tests needs to run as priviliged/root otherwise classic file test doesn't have permission to the PV
            # k8s-netperf needs privileged to use host-network
            oc label --overwrite ns ${test_namespace} pod-security.kubernetes.io/warn=privileged
            oc adm policy add-scc-to-user privileged -n ${test_namespace} -z default
        fi
        if [[ ${test_namespace} == incluster-apiserver ]]; then
            # Incluster-apiserver has multiple service accounts - so need to add permission to them all
            # We can't look these up dynamically as they don't exist at this point - and they have to be created before the pods are created.
            instances=10
            for i in $(seq 0 ${instances}); do
                oc adm policy add-scc-to-user nonroot -n ${test_namespace} -z test${i}-incluster-apiserver-sa
            done
        fi
        if [[ ${test_namespace} == k8s-e2e-performance* ]]; then
            # Clusterloader2 tests need anyuid in monitoring and probes namespace - the namespaces need to exist before doing this
            set +e
            oc get namespace monitoring >/dev/null 2>&1
            if [[ $? -ne 0 ]]; then
                oc create namespace monitoring
            fi
            oc get namespace probes >/dev/null 2>&1
            if [[ $? -ne 0 ]]; then
                oc create namespace probes
            fi
            set -e
            oc adm policy add-scc-to-user anyuid -n monitoring -z prometheus-k8s
            oc adm policy add-scc-to-user anyuid -n monitoring -z grafana
            oc adm policy add-scc-to-user anyuid -n monitoring -z prometheus-operator
            oc adm policy add-scc-to-user anyuid -n monitoring -z default
            oc adm policy add-scc-to-user anyuid -n probes -z default
            oc adm policy add-scc-to-user hostaccess -n probes -z default
        fi
    else
        echo "ERROR: Failed to enable root user in openshift pods"
    fi
    kubectl config use-context ${clusterName}
fi
set -e
