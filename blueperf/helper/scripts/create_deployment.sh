#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2018, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

set -e
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
  echo "usage: $0 [-c|--cluster] [-i|--imagename] [-y|--yamlfile] [-d|--directory] [-ing|--ingress] [-r|--route]"
  echo "  -c      Cluster Name"
  echo "  -i      Image Name"
  echo "  -y      YAML File Name"
  echo "  -d      Directory Name"
  echo "  -r      (Optional) Set true to Create Openshift route"
  echo "  -ing    (Optional) Set true to Create Ingress Controller"
  exit 1
}

[ -z $1 ] && { usage; }

IMAGENAME=
YAML_FILE=
CLUSTER_NAME=
INGRESS_URL=
INGRESS=false
ROUTE=false

export IKS_BETA_VERSION=1

# point to a temporary kubeconfig so that we don't keep growing on ~/.kube/config each time
export KUBECONFIG=$(mktemp --suffix=.yml)
touch $KUBECONFIG
# and make sure it gets cleaned up on exit
function cleanupconfig() {
  rm -f $KUBECONFIG
}
trap cleanupconfig EXIT

while true; do
  case "$1" in
  -c | --cluster)
    CLUSTER_NAME="$2"
    shift 2
    ;;
  -i | --imagename)
    IMAGENAME="$2"
    shift 2
    ;;
  -y | --yamlfile)
    YAML_FILE="$2"
    shift 2
    ;;
  -d | --directory)
    DIRECTORY="$2"
    shift 2
    ;;
  -r | --route)
    ROUTE="$2"
    shift 2
    ;;
  -ing | --ingress)
    INGRESS="$2"
    shift 2
    ;;
  --)
    shift
    break
    ;;
  *) break ;;
  esac
done

if [[ -z "${CLUSTER_NAME// /}" ]]; then
  echo "${yel}No cluster name provided. Will try to get an existing cluster...${end}"
  # This will not work as last line of ibmcloud ks clusters may not give you a cluster
  # We always run with cluster name so leaving it as it
  CLUSTER_NAME=$(ibmcloud ks clusters | tail -1 | awk '{print $1}')

  if [[ "$CLUSTER_NAME" == "Name" ]]; then
    echo "No Kubernetes Clusters exist in your account. Please provision one and then run this script again."
    exit 1
  fi
fi
# Getting Cluster Configuration
echo "${grn}Getting configuration for cluster ${CLUSTER_NAME}...${end}"

# Retry command - see https://github.ibm.com/alchemy-containers/armada-performance/issues/2511
  set +e
  retries=3
  counter=1

  until [[ ${counter} -gt ${retries} ]]; do
  
      if [[ ${counter} -gt 1 ]]; then
          printf "%s - %d. Command failed. Retrying in 5s \n" "$(date +%T)" "${counter}"
          sleep 5
      fi

      ibmcloud ks cluster config --cluster ${CLUSTER_NAME} --admin

      if [[ $? == 0 ]]; then
         # Command was successful so exit with no more retries
         break
      fi

      if [[ ${counter} == 3 ]]; then
         # Exiting as we hit the maximum number of retries
         printf "Command failed. Hit maximum number of retries \n"
         exit 1
      fi

      ((counter++))
  done
  set -e

eval INGRESS_URL="$(ibmcloud ks cluster get --cluster ${CLUSTER_NAME} | grep "Ingress Subdomain" | awk '{print $3}')"
ROUTE_HOST="acmeair.${INGRESS_URL}"

cd ./${DIRECTORY}

# Using manifests files from git clone
if [ "$ROUTE" = true ]; then
  MANIFESTS="manifests-openshift"
else
  MANIFESTS="manifests"
fi

# DIRECTORY is acmeair-<APP>-java when route is true
APP=$(echo ${DIRECTORY} | sed "s#acmeair-##" | sed "s#-java##")
echo "Deploying ${APP} application"
APP_YAML="${MANIFESTS}/deploy-acmeair-${APP}-java.yaml"
PATCHED_APP_YAML="${MANIFESTS}/patched_deploy-acmeair-${APP}-java.yaml"
ROUTE_YAML="${MANIFESTS}/acmeair-${APP}-route.yaml"
PATCHED_ROUTE_YAML="${MANIFESTS}/patched_acmeair-${APP}-route.yaml"

printf "${grn}Patching ${APP_YAML} with image ${IMAGENAME}${end}\n"
sed "s#${DIRECTORY}:latest#${IMAGENAME}#g" ${APP_YAML} >${PATCHED_APP_YAML}
printf "${grn}Creating Kubernetes Deployment${end}\n"
kubectl create -f ${PATCHED_APP_YAML}

# Using manifests files from git clone
if [ "$ROUTE" = true ]; then
  printf "${grn}Patching ${ROUTE_YAML} with ${ROUTE_HOST}${end}\n"
  sed "s#_HOST_#${ROUTE_HOST}#g" ${ROUTE_YAML} >${PATCHED_ROUTE_YAML}
  printf "${blu}Creating Route${end}\n"
  kubectl create -f ${PATCHED_ROUTE_YAML}
elif [ "$INGRESS" = true ]; then
  ING_YAML="../scripts/ing.yaml"
  INGTEMP_YAML="../scripts/ingtemp.yaml"
  eval INGRESS_URL="$(ibmcloud ks cluster get --cluster ${CLUSTER_NAME} | grep "Ingress Subdomain" | awk '{print $3}')"
  printf "${grn}Generating Temp Ingress yaml file for ing.yaml with ${INGRESS_URL}${end}\n"
  sed "s#INGRESS_URL#${INGRESS_URL}#g" ${ING_YAML} >${INGTEMP_YAML}
  printf "${blu}Creating Ingress Controller${end}\n"
  kubectl create -f ${INGTEMP_YAML}
  printf "${blu}Deleting Temp Ingress yaml file${end}\n"
  rm ${INGTEMP_YAML}
fi
