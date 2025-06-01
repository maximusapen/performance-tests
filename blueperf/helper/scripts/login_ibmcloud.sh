#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018, 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

set -e

export IKS_BETA_VERSION=1

# point to a temporary kubeconfig so that we don't keep growing on ~/.kube/config each time
export KUBECONFIG=$(mktemp --suffix=.yml)
touch $KUBECONFIG
# and make sure it gets cleaned up on exit
function cleanupconfig() {
    rm -f $KUBECONFIG
}
trap cleanupconfig EXIT

# Terminal Colors
red=$'\e[1;31m'
grn=$'\e[1;32m'
yel=$'\e[1;33m'
blu=$'\e[1;34m'
mag=$'\e[1;35m'
cyn=$'\e[1;36m'
end=$'\e[0m'
coffee=$'\xE2\x98\x95'
coffee3="${coffee} ${coffee} ${coffee}"

function usage() {
    echo "usage: $0 [-a|--akipey] [-c|--cluster] [-s|--space] [-r|--region]"
    echo "  -a      API KEY for Bluemix"
    echo "  -c      Cluster Name"
    echo "  -s      (Optional) Space Name - Default : dev"
    echo "  -r      (Optional) Region Name - Default : ng"
    exit 1
}

[ -z $1 ] && { usage; }

SPACE=
REGION="us-south"
CLUSTER_NAME=
PERF_METADATA_TOML=/performance/armada-perf/armada-perf-client2/config/perf-metadata.toml

# Get the correct endpoints for env used
# There are multiple endpoints defined in the toml and we need the iks one so make sure we get the correct one by looking at the parent
IKS_ENDPOINT=$(/performance/bin/tomlToJson $PERF_METADATA_TOML | jq -r '.iks.endpoint')
ENVIRONMENT=$(/performance/bin/tomlToJson $PERF_METADATA_TOML | jq -r '.ibmcloud.iam_endpoint' | cut -d '.' -f2-)

while true; do
    case "$1" in
    -a | --apikey)
        API_KEY="$2"
        shift 2
        ;;
    -c | --cluster)
        CLUSTER_NAME="$2"
        shift 2
        ;;
    -s | --space)
        SPACE="$2"
        shift 2
        ;;
    -r | --region)
        REGION="$2"
        shift 2
        ;;
    --)
        shift
        break
        ;;
    *) break ;;
    esac
done

API_ENDPOINT=${ENVIRONMENT}

printf "${grn}API Endpoint is set to $API_ENDPOINT${end}\n"

ibmcloud plugin update container-service -r "IBM Cloud" -f
ibmcloud plugin update container-registry -r "IBM Cloud" -f

# IBM Cloud Login
printf "${grn}Logging into IBM Cloud${end}\n"
if [[ -z "${API_KEY// /}" && -z "${SPACE// /}" ]]; then
    echo "${yel}API Key & SPACE NOT provided.${end}"
    ibmcloud login -a ${API_ENDPOINT} -r ${REGION}

elif [[ -z "${SPACE// /}" ]]; then
    echo "${yel}API Key provided but SPACE was NOT provided.${end}"
    export IBMCLOUD_API_KEY=${API_KEY}
    ibmcloud login -a ${API_ENDPOINT} -r ${REGION}

elif [[ -z "${API_KEY// /}" ]]; then
    echo "${yel}API Key NOT provided but SPACE was provided.${end}"
    ibmcloud login -a ${API_ENDPOINT} -s ${SPACE}

else
    echo "${yel}API Key and SPACE provided.${end}"
    export IBMCLOUD_API_KEY=${API_KEY}
    ibmcloud login -a ${API_ENDPOINT} -s ${SPACE} -o armada.performance
fi

if [[ -z "${CLUSTER_NAME// /}" ]]; then
    echo "${yel}No cluster name provided. Will try to get an existing cluster...${end}"
    # This will not work as last line of ibmcloud ks clusters may not give you a cluster
    # We always run with cluster name so leaving it as is
    CLUSTER_NAME=$(ibmcloud ks clusters | tail -1 | awk '{print $1}')

    if [[ "$CLUSTER_NAME" == "Name" ]]; then
        echo "No Kubernetes Clusters exist in your account. Please provision one and then run this script again."
        exit 1
    fi
fi

# Getting Cluster Configuration

printf "\n\n${grn}Login into IBM Kubernetes Service${end}\n\n"
ibmcloud ks init --host ${IKS_ENDPOINT}

echo "${grn}Getting configuration for cluster ${CLUSTER_NAME}...${end}"
ibmcloud ks cluster config --cluster ${CLUSTER_NAME}

#printf "\n\n${grn}Getting Account Information...${end}\n"
#ORG=$(cat ~/.bluemix/.cf/config.json | jq .OrganizationFields.Name | sed 's/"//g')
#SPACE=$(cat ~/.bluemix/.cf/config.json | jq .SpaceFields.Name | sed 's/"//g')

# Creating for API KEY
if [[ -z "${API_KEY// /}" ]]; then
    printf "\n\n${grn}Creating API KEY...${end}\n"
    # Last line of ibmcloud iam api-key-create kubekey is UUID.  Two lines above is API Key.
    # If problem with this key, try to use API Key
    API_KEY=$(ibmcloud iam api-key-create kubekey | tail -1 | awk '{print $3}')
    echo "${yel}API key 'kubekey' was created.${end}"
    echo "${mag}Please preserve the API key! It cannot be retrieved after it's created.${end}"
    echo "${cyn}Name${end}	kubekey"
    echo "${cyn}API Key${end}	${API_KEY}"
fi

printf "\n\n${grn}Login into Container Registry${end}\n\n"
ibmcloud cr api
ibmcloud cr login
