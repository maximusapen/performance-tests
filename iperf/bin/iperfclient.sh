#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018, 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

#Defaults
registry=""
environment=""
address=""
namespace="default"
pods=1
id=1
load_balancer=false

deploymentName=iperfclient

while test $# -gt 0; do
    case "$1" in
    -h | --help)
        echo "iperfclient.sh - runs kubernetes based Linux iperf client"
        echo " "
        echo "iperfclient.sh [options] args"
        echo " "
        echo "options:"
        echo "-h, --help                        show brief help"
        echo "-e, --environment environment     Registry environment namespace (e.g dev7, stage1, etc.)"
        echo "-g, --registry registry_url       image registry location"
        echo "-d, --deployment chart_name       helm chart deployment name"
        echo "-n, --namespace k8s_namespace     kubernetes namespace for deployment"
        echo "-i, --id identifier               specify an identifier to use (allows multiple deployments in 1 cluster) - must be an integer"
        echo "-a, --address                     The IP address of the iperfserver to connect to"
        echo "-p, --port                        The port of the iperfserver to connect to"
        echo "-l, --loadbalancer                use the load balancer"
        exit 0
        ;;
    -i | --id)
        shift
        if test $# -gt 0; then
            if ! [[ "$1" =~ ^[0-9]+$ ]]; then
                echo "ID must be an integer"
                exit 1
            fi
            id=$1
        else
            echo "ID not specified"
            exit 1
        fi
        shift
        ;;
    -a | --address)
        shift
        if test $# -gt 0; then
            address=$1
        else
            echo "Address not specified"
            exit 1
        fi
        shift
        ;;
    -p | --port)
        shift
        if test $# -gt 0; then
            port=$1
        else
            echo "Port not specified"
            exit 1
        fi
        shift
        ;;
    -n | --namespace)
        shift
        if test $# -gt 0; then
            namespace=$1
        else
            echo "Namespace not specified"
            exit 1
        fi
        shift
        ;;
    -d | --deployment)
        shift
        if test $# -gt 0; then
            deploymentName=$1
        else
            echo "Helm chart deployment name not specified"
            exit 1
        fi
        shift
        ;;
    -e | --environment)
        shift
        if test $# -gt 0; then
            environment=$1
        else
            echo "registry namespace environment not specified"
            exit 1
        fi
        shift
        ;;
    -g | --registry)
        shift
        if test $# -gt 0; then
            registry=$1
        else
            echo "registry url not specified"
            exit 1
        fi
        shift
        ;;
    -l | --loadbalancer)
        load_balancer=true
        shift
        ;;
    *)
        args="${args} "$*
        break
        ;;
    esac
done

echo Chart: ${deploymentName}
echo Namespace: ${namespace}
if [ -z ${port} ]; then
    if [ "$load_balancer" = true ]; then
        let "port = ${id} + 40520"
    else
        let "port = ${id} + 30520"
    fi
fi
#port=32926
echo Port: ${port}
echo id: ${id}

if [[ -n ${address} ]]; then
    echo Address: ${address}
else
    echo "-a or --address must be specified"
    exit 1
fi

# This is a client - so always set -c & port
args="${args} -c ${address} -p ${port}"
echo Args: ${args}

if [[ -n ${registry} ]]; then
    echo Registry: ${registry}
    setRegistry="--set image.registry=${registry}"
fi

if [[ -n ${environment} ]]; then
    echo Registry Environment: ${environment}
    setEnvironment="--set image.name=armada_performance_${environment}/iperf"
fi

# Delete any existing job from previous runs
helm uninstall "${deploymentName}-${id}" --namespace ${namespace} 2>/dev/null

# Create a new config map so that we can pass in the supplied parameters to the pod creation
kubectl delete configmap "${deploymentName}-${id}-config" --ignore-not-found=true --namespace=${namespace}

sleep 10
kubectl create configmap "${deploymentName}-${id}-config" --namespace=${namespace} --from-literal=PERF_IPERF_ARGS="${args}"

# Use helm to install the chart and execute the tests as a kubernetes job on the cluster
helm install "${deploymentName}-${id}" ../imageDeploy/iperfclient --namespace=${namespace} --set port=${port} --set id=${id} ${setCPU} ${setVMBytes} ${setRegistry} ${setEnvironment}
