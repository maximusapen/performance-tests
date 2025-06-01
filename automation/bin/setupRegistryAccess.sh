#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2017, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# This script will ensure that clusters have access to pull images from the IBM Cloud Registry
# for the specified Kubernetes namespace.
# It uses the "armada-perf-read-reg" service-id, which has read access to our registry.
# See https://cloud.ibm.com/docs/Registry?topic=registry-registry_access#registry_access_serviceid_apikey
#
# Input Paramters:
# 1. Kubernetes namespace (required)
# 2. Flag to indicate whether the specified namespace should be (re)created (optional - default true)
# 3. Flag to indicate whether the secret should be recreated if it already exists (optional - default false)


patch_with_retry() {
   patch_command=$1

    set +e
    local retries=6
    local counter=1

    # Support retry of patch command which sometimes has timing issues
    until [[ ${counter} -gt ${retries} ]]; do
        if [[ ${counter} -gt 1 ]]; then
            printf "%s - %d. Patch command failed. Retrying.\n" "$(date +%T)" "${counter}"
        fi

        eval ${patch_command}

        if [[ $? == 0 ]]; then
            # Patch command was successful
            return 0
        fi

        sleep 10
        ((counter++))
    done
    set -e
    return 1
}


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

create_namespace=true
if [[ -n $2 ]]; then
    create_namespace=$2
fi

recreate_secret=false
if [[ -n $3 ]]; then
    recreate_secret=$3
fi

registryName="stg.icr.io"
userEmail="armada.performance@uk.ibm.com"
registrySecretName="perf-stg-icr-io"

printf "Authorizing Kubernetes namespace '%s' for use with IBM Cloud Registry.\n" "${test_namespace}"

# Create the specified namespace, deleting it first if it already exists
if [[ "${create_namespace}" == true ]]; then
    if [[ "${test_namespace}" != "default" ]]; then
        kubectl delete namespace "${test_namespace}" --ignore-not-found=true --grace-period=15
        kubectl create namespace "${test_namespace}"
    fi
fi

# If not already done, create a secret in the default namespace to enable registry read access
perfRegScrt=$(kubectl get secret -n default ${registrySecretName} --no-headers --ignore-not-found=true)
if [[ -z ${perfRegScrt} || ( "${recreate_secret}" == true && "${test_namespace}" == "default" ) ]]; then
    if [[ -z "${STAGE_GLOBAL_ARMPERF_REGISTRY_APIKEY}" ]]; then
        printf "Cannot create secret for cluster access to our registry. Service ID api key not set"
        exit 1
    fi

    if [[ -n ${perfRegScrt} && "${recreate_secret}" == true ]]; then
        kubectl delete secret -n default ${registrySecretName} --ignore-not-found=true
    fi
    printf "Creating registry access secret in default namespace"
    kubectl create secret -n default docker-registry ${registrySecretName} --docker-server=${registryName} --docker-username=iamapikey --docker-password="${STAGE_GLOBAL_ARMPERF_REGISTRY_APIKEY}" --docker-email=${userEmail} # pragma: allowlist secret
fi

# ...and then copy it to the specified namespace if not using the default namespace
if [[ "${test_namespace}" != "default" ]]; then
    if [[ "${recreate_secret}" == true ]]; then
        kubectl delete secret -n "${test_namespace}" ${registrySecretName} --ignore-not-found=true
    fi

    printf "Copying registry access secret '%s' to '%s' namespace\n" "${registrySecretName}" "${test_namespace}"
    kubectl -n default get secret ${registrySecretName} -o yaml | sed "s/default/${test_namespace}/g" | kubectl -n "${test_namespace}" create -f -

    # Check the secrets were created successfully
    secretCount=$(kubectl get secret -n "${test_namespace}" ${registrySecretName} --no-headers --ignore-not-found | wc -l)

    if [[ ${secretCount} -eq 0 ]]; then
        printf "Failed to create IBM Container Registry secrets for Kubernetes '%s' namespace\n" "${test_namespace}"
        exit 1
    fi
fi

# Finally add as an image pull secret in the default service account for the specifed namespace
printf "Adding imagePullSecret to default service account in '%s'  & default namespace\n" "${test_namespace}"

patch_with_retry "kubectl patch -n \"${test_namespace}\" serviceaccount/default -p \"{\\\"imagePullSecrets\\\":[{\\\"name\\\": \\\"${registrySecretName}\\\"}]}\""
patch_with_retry "kubectl patch serviceaccount/default -p \"{\\\"imagePullSecrets\\\":[{\\\"name\\\": \\\"${registrySecretName}\\\"}]}\""
