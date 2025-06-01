#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020, 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# This script will login to the Openshift server and save the login for subsequent use

# KUBECONFIG environment must be set
# ARMADA_PERFORMANCE_API_KEY must be set with the approppriate API Key

if [[ -z "${KUBECONFIG}" ]]; then
    printf "KUBECONFIG not set. Exiting.\n"
    exit 1
fi

# Controls whether or not the the server's certificate will not be checked for validity
# For Satellite clusters in stage, a self signed certificate is used, and thus we need to skip the check.
oc_insecure_login=$1
if [[ -z "${oc_insecure_login}" ]]; then
    oc_insecure_login=false
fi

echo "Login to openshift cluster"
server=$(grep server: $KUBECONFIG | awk '{print $2}' | head -1)
if [[ -n ${ARMADA_PERFORMANCE_API_KEY} ]]; then
    # NOTE: If this call returns 500 error message then the calling script must run and give write permission to the config
    #       ${perf_dir}/bin/armada-perf-client --action=GetClusterConfig --clusterName="${load_cluster_name}"
    oc login -u apikey -p ${ARMADA_PERFORMANCE_API_KEY} --server=${server} --insecure-skip-tls-verify=${oc_insecure_login}
else
    echo "Unable to login to Openshift as ARMADA_PERFORMANCE_API_KEY is not set."
fi
