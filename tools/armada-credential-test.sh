#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# Script used for testing issue https://github.ibm.com/alchemy-containers/armada-performance/issues/2326
#
# It takes a number as the only parameter, and will create this many service IDs and service API-Keys. 
# It will then login, and get the admin cluster config from every cluster on the carrier. 
# It is expected this will run on carrier5 where there will be a large number of clusters.

# It requires a valid API-Key to be set as the Environment variable ARMADA_PERFORMANCE_API_KEY before the script is run.
# Note the API-Key can be a personal API-Key - that name was used for consistency across other scripts, and in case this is to be 
# run as a Jenkins job using the functional ID.

if [[ -z "$1" ]]; then
    echo "Please specify the number of users to run with"
    exit 1
fi

if [[ -z "$ARMADA_PERFORMANCE_API_KEY" ]]; then
    echo "Please ensure ARMADA_PERFORMANCE_API_KEY environment variable is set before running"
    exit 1
fi
num_users=$1

perf_dir=/performance

# Need to login to IBM Cloud so we can use the ibmcloud is commands
PERF_METADATA_TOML=${perf_dir}/armada-perf/armada-perf-client2/config/perf-metadata.toml
IKS_ENDPOINT=$(/performance/bin/tomlToJson $PERF_METADATA_TOML | jq -r '.iks.endpoint')
API_ENDPOINT=$(/performance/bin/tomlToJson $PERF_METADATA_TOML | jq -r '.ibmcloud.iam_endpoint' | cut -d '.' -f2-)
REGION="us-south"

ibmcloud plugin install container-service -r "IBM Cloud" -f
ibmcloud plugin update container-service -r "IBM Cloud" -f

ibmcloud config --check-version=false

export IBMCLOUD_API_KEY=${ARMADA_PERFORMANCE_API_KEY}
ibmcloud login -a ${API_ENDPOINT} -r ${REGION}

printf "\n\n${grn}Login into IBM Kubernetes Service${end}\n\n"
ibmcloud ks init --host ${IKS_ENDPOINT}

echo "Getting list of clusters on the carrier"
cluster_list_file_name="cluster_list.txt"
ibmcloud ks clusters | grep -E "normal|warning" | grep -v openshift | awk '{print $1}' > ${cluster_list_file_name}
cluster_count=$(cat ${cluster_list_file_name} | wc -l)
echo "Each user will get config for ${cluster_count} clusters"

for ((userNum = 1; userNum <= ${num_users}; userNum++)); do
    ibmcloud login --no-region --apikey ${ARMADA_PERFORMANCE_API_KEY}
    user="testuser$userNum"
    echo "Creating Service ID for user ${user}"
    ibmcloud iam service-id-create ${user}

    echo "Creating Service Policy for user ${user}"
    ibmcloud iam service-policy-create ${user} --roles Administrator,Manager --service-name containers-kubernetes

    apikey_name="${user}-apikey"
    echo "Creating API Key ${apikey_name} for user ${user}"
    service_apikey=$(ibmcloud iam service-api-key-create ${apikey_name} ${user} -d "Testing for armada-credential - DM" --output=json | jq -j '.apikey')
    echo "Logging in as user ${user}"
    ibmcloud login --no-region --apikey ${service_apikey}
    ibmcloud ks init --host ${IKS_ENDPOINT}

    for clusterName in `cat  ${cluster_list_file_name}`; do 
        echo "Getting admin cluster config for ${clusterName} using user ${user}"
        ibmcloud ks cluster config --cluster ${clusterName} --admin
    done

    # Cleanup the user at the end
    echo "Deleting Service ID for user ${user}"
    ibmcloud login --no-region --apikey ${ARMADA_PERFORMANCE_API_KEY}
    ibmcloud iam service-id-delete ${user} -f

done

rm ${cluster_list_file_name}
