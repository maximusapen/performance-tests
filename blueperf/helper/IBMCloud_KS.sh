#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

set -e
##################################################
####ENTER THESE VARIABLES TO USE THIS SCRIPT####
##################################################
# Position Parameters
# 1. cluster name
CLUSTER_NAME=
#Default Dallas Region, dev space
REGION="us-south"
SPACE=dev
REGISTRY=stg.icr.io
NAMESPACE=armada_performance
DOCKER_PASSWORD=
IMAGE_TAG=latest

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
  echo "usage:"
  echo "${cyn}To Install CLI and plugins${end}" $0 " [-p|--prep]"
  echo "e.g.${grn} $0 -p${end}"
  echo "${cyn}To Login IBM Cloud and Container Service${end}" $0 " [-l|--login]"
  echo "e.g.${grn} $0 -l${end}"
  echo "${cyn}To Clone git repository${end}" $0 " [-c|--clone]"
  echo "e.g.${grn} $0 -c${end}"
  echo "${cyn}To Deploy Containers to the Container Service${end}" $0 " [-d|--deploy]"
  echo "e.g.${grn} $0 -d${end}"
  echo "${cyn}To Populate DB${end}" $0 " [-db|--loaddb]"
  echo "e.g.${grn} $0 -db${end}"
  echo "${cyn}To Undeploy Containers from the Container Service${end}" $0 " [-u|--undeploy]"
  echo "e.g.${grn} $0 -u${end}"
  echo "${cyn}To Login, Clone, Deploy and Populate DB NOTE:It will pause 2 minutes before DB polulation. If DB is not populated, run -db separately${end}" $0 " [-a|--all]"
  echo "e.g.${grn} $0 -a${end}"
  echo "${cyn}To Enable Istio deployment${end}" $0 " [-istio|--istio]"
  echo "e.g.${grn} $0 -a${end}"
  echo "${cyn}To create image and upload to IBM Registry${end}" $0 " [-image|--image]"
  echo "e.g.${grn} $0 -a${end}"
  echo "${cyn}To use route${end}" $0 " [-r|--route]"
  echo "e.g.${grn} $0 -a${end}"

  echo "Details of all options:"
  echo "  -a      Run below -l, -c, -d and -db commands"
  echo "  -p      Install CLI and plugins"
  echo "  -l      Login IBM Cloud and Container Service"
  echo "  -c      Clone git repositories"
  echo "  -d      Deploy Containers"
  echo "  -db     Load DB"
  echo "  -u      Undeploy Containers"
  echo "  -cl     Cluster name"
  echo "  -r      Use route"
  echo "  -istio  Enable Istio deployment"
  echo "  -image  Create image and upload to IBM Registry"
  exit 1
}

[ -z $1 ] && { usage; }

LOGIN=false
CLI=false
CLONE=false
DEPLOY=false
UNDEPLOY=false
INGRESS=false
DB=false
PAUSE=false
ISTIO=false
ROUTE=false
IMAGE=false

# Branch specific data
BLUEPERF_BRANCH="microprofile-3.2"
MANIFESTS="manifests"
IMAGE_EXT="-${BLUEPERF_BRANCH}"
#DOCKERFILE is used when using the Dockerfile in blueperf/helper/dockerfile
DOCKERFILE="Dockerfile-base"

while true; do
  case "$1" in
  -a | --all)
    LOGIN=true
    CLONE=true
    DEPLOY=true
    DB=true
    PAUSE=true
    shift
    ;;
  -p | --prep)
    CLI=true
    shift
    ;;
  -l | --login)
    LOGIN=true
    shift
    ;;
  -d | --deploy)
    DEPLOY=true
    shift
    ;;
  -db | --loaddb)
    DB=true
    shift
    ;;
  -u | --undeploy)
    UNDEPLOY=true
    shift
    ;;
  -c | --clone)
    CLONE=true
    shift
    ;;
  -r | --route)
    ROUTE=true
    MANIFESTS="manifests-openshift"
    shift
    ;;
  -cl | --cluster)
    shift
    if test $# -gt 0; then
      CLUSTER_NAME=$1
    else
      echo "cluster name not specified"
      exit 1
    fi
    shift
    ;;
  -pause | --pause)
    PAUSE=true
    shift
    ;;
  -istio | --istio)
    ISTIO=true
    shift
    ;;
  -image | --image)
    IMAGE=true
    CLONE=true
    shift
    ;;
  --)
    shift
    break
    ;;
  *) break ;;
  esac
done

printf "Region : ${cyn}$REGION${end}\n"
printf "Cluster : ${cyn}$CLUSTER_NAME${end}\n"
printf "Namespace : ${cyn}$NAMESPACE${end}\n"
printf "ISTIO : ${cyn}$ISTIO${end}\n"
printf "IMAGE_EXT : ${cyn}$IMAGE_EXT${end}\n"

if [ "$CLI" = true ]; then
  printf "${grn}Running install_cli.sh${end}\n"
  ./scripts/install_cli.sh
fi
if [[ "$LOGIN" == true ]]; then
  printf "${grn}Running login_ibmcloud.sh with apikey ${REGION}${end}\n"
  ./scripts/login_ibmcloud.sh -c ${CLUSTER_NAME} -r ${REGION}
fi

declare -a arr=("acmeair-authservice-java" "acmeair-bookingservice-java" "acmeair-customerservice-java" "acmeair-flightservice-java")

if [[ "$CLONE" == true ]]; then
  printf "${cyn}Cloning Acmair Homogeneous Java Microservice${end}\n"
  for i in "${arr[@]}"; do
    if [[ "$IMAGE" == true ]]; then
      # If we are creating image, we always clone the branch
      if [ -d "$i" ]; then
        # Move directory, if exists to lastBranches in case we need to re-instate the branch
        mkdir -p lastBranches
        rm -fr lastBranches/*
        mv "$i" lastBranches
      fi
    fi
    if [ ! -d "$i" ]; then
      echo Cloning "$i" branch ${BLUEPERF_BRANCH}
      git clone https://github.com/blueperf/${i} -b ${BLUEPERF_BRANCH}
    else
      echo Directory "$i" exists. Checking branch.
      cd "$i"
      git status | grep "On branch"
      set +e
      branch=$(git status | grep "On branch ${BLUEPERF_BRANCH}")
      set -e
      if [ "$branch" == "" ]; then
        echo Branch is not ${BLUEPERF_BRANCH}. Switch.
        git checkout ${BLUEPERF_BRANCH}
      else
        echo Same branch as ${BLUEPERF_BRANCH}
      fi
      git status | grep "On branch"
      cd ..
    fi
  done
fi

if [[ "$DEPLOY" == true ]]; then
  for i in "${arr[@]}"; do
    printf "${cyn}Using Image ${i} in ${REGISTRY}${end}\n"
    YAML_FILE=${i}.yaml
    DIRECTORY=${i}
    if [[ ${REGISTRY} == *"icr.io"* ]]; then
      IMAGE_NAME=${REGISTRY}/${NAMESPACE}/${i}${IMAGE_EXT}:${IMAGE_TAG}
    else
      IMAGE_NAME=${REGISTRY}/${i}${IMAGE_EXT}:${IMAGE_TAG}
      docker login -u ${REGISTRY} -p ${DOCKER_PASSWORD}
    fi
    echo IMAGE_NAME: ${IMAGE_NAME}
    INGRESS=false
    if [[ ${i} == *"auth"* && ${ISTIO} == false ]]; then
      printf "${grn}Also generating Ingress${end}\n"
      if [[ ${ROUTE} == false ]]; then
        INGRESS=true
      fi
    fi

    # No need to create acmeair image and upload to registry for every test as we are testing IKS/istio, not acmeair.
    # A separate job to create image will pass in -image/--image to create image and continue to test the image.

    if [[ "${IMAGE}" == true ]]; then
      # Blueperf script create_image.sh uploaded image to IBM registry.
      # Use our own version of Dockfile in dockerfile directory for the app.
      #echo Creating ${IMAGE_NAME} using ${i}/Dockerfile
      echo Creating ${IMAGE_NAME} using ../dockerfile/${DOCKERFILE}-${i}

      # This command uses Docker files in blueperf GIT
      #./scripts/create_image.sh -i ${IMAGE_NAME} -d ${i} -f "Dockerfile"

      # This command uses Docker files in blueper/helper/dockerfile in armada-performance GIT
      ./scripts/create_image.sh -i ${IMAGE_NAME} -d ${i} -f ../dockerfile/${DOCKERFILE}-${i}
    fi

    CREATE_DEPLOYMENT_DIR="${i}"
    ./scripts/create_deployment.sh -c ${CLUSTER_NAME} -i ${IMAGE_NAME} -d ${CREATE_DEPLOYMENT_DIR} -y deploy-${i}.yaml -r ${ROUTE} -ing ${INGRESS}

    if [[ "${IMAGE}" == true ]]; then
      # Delete local image so deployment always download image from registry to pick up latest image for next run
      docker image rm ${IMAGE_NAME}
    fi
  done
fi

if [[ "${IMAGE}" == true ]]; then
  # Remove maven cache as not needed after the build
  rm -Rf ~/.m2/repository/*
fi

if [[ "$PAUSE" == true ]]; then
  printf "${grn}Pausing 4 minutes for Pods to be created & Liberty Server to startup${end}\n"
  sleep 240
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

  kubectl get deploy | grep acmeair
  printf "%s - Printing Liberty/Java version for flight service: \n" "$(date +%T)"
  kubectl logs -l  name=acmeair-flight-deployment --tail=100 | grep Launching
  set -e
fi

if [[ "$DB" == true ]]; then
  eval INGRESS_URL="$(ibmcloud ks cluster get --cluster ${CLUSTER_NAME} | grep "Ingress Subdomain" | awk '{print $3}')"

  if [[ "${ISTIO}" == true || "${ROUTE}" == true ]]; then
    INGRESS_URL="acmeair.${INGRESS_URL}"
  fi
  echo INGRESS_URL: ${INGRESS_URL}

  curl http://${INGRESS_URL}/booking/loader/load || true
  printf "\n"
  curl http://${INGRESS_URL}/flight/loader/load || true
  printf "\n"
  curl http://${INGRESS_URL}/customer/loader/load?numCustomers=10000 || true
  printf "\n"
  printf "${grn}Database Loaded${end}\n"
fi

if [[ "$UNDEPLOY" == true ]]; then
  # Getting Cluster Configuration
  echo "${grn}Getting configuration for cluster ${CLUSTER_NAME}...${end}"
  ibmcloud ks cluster config --cluster ${CLUSTER_NAME} --admin
  # Deployments can be deleted from previous runs but files still exists on client.
  echo Ignore errors if deployments not found.
  kubectl delete --ignore-not-found=true -f scripts/ing.yaml || true
  for i in "${arr[@]}"; do
    kubectl delete --ignore-not-found=true -f ${i}/${MANIFESTS}/deploy-${i}.yaml || true
  done
fi
