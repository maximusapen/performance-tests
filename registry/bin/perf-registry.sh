#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

#Defaults
registry=""
regional=""
namespace="default"
verbose=false
metrics=false
registrykey=""
registryIp=""
international=false
allRegions=false
clusterRegion=""
hyperkubeImage=""

jobName=perf-registry

while test $# -gt 0; do
    case "$1" in
    -h | --help)
        echo "perf-registry - runs kubernetes based registry performance tests"
        echo " "
        echo "perf-registry [options]"
        echo " "
        echo "options:"
        echo "-h, --help                	show brief help"
        echo "-k, --registrykey key         IBM registry access api-key"
        echo "-e, --environment environment     registry environment namespace (e.g dev7, stage1, etc.)"
        echo "-g, --registry registry_url 	image registry location"
        echo "-r, --regional registry_url 	regional registry location"
        echo "-n, --namespace k8s_namespace     kubernetes namespace for deployment"
        echo "-m, --metrics             	send results to metrics service"
        echo "-v, --verbose             	log test results to stdout"
        echo "-i, --international        	run against international registry"
        echo "-l, --allRegions           	run against all regional registries from one cluster"
        echo "-c, --clusterRegion        	cluster region to use if --allRegions specified"
        echo "-y, --hyperkubeImage       	hyperkube image name to pull"
        exit 0
        ;;
    -l | --allRegions)
        allRegions=true
        ;;
    -c | --clusterRegion)
        shift
        if test $# -gt 0; then
            clusterRegion=$1
        else
            echo "cluster region not specified"
            exit 1
        fi
        ;;
    -i | --international)
        international=true
        ;;
    -v | --verbose)
        verbose=true
        ;;
    -m | --metrics)
        metrics=true
        ;;
    -g | --registry)
        shift
        if test $# -gt 0; then
            registry=$1
        else
            echo "registry url not specified"
            exit 1
        fi
        ;;
    -r | --regional)
        shift
        if test $# -gt 0; then
            regional=$1
        else
            echo "regional registry url not specified"
            exit 1
        fi
        ;;
    -n | --namespace)
        shift
        if test $# -gt 0; then
            namespace=$1
        else
            echo "Namespace not specified"
            exit 1
        fi
        ;;
    -e | --environment)
        shift
        if test $# -gt 0; then
            environment=$1
        else
            echo "registry namespace environment not specified"
            exit 1
        fi
        ;;
    -k | --registrykey)
        shift
        if test $# -gt 0; then
            registrykey=$1
        fi
        ;;
    -y | --hyperkubeImage)
        shift
        if test $# -gt 0; then
            hyperkubeImage=$1
        else
            echo "Hyperkube image not specified"
            exit 1
        fi
        ;;
    -dns)
        shift
        if test $# -gt 0; then
            registryDNS=$1
        else
            echo "Registry dns not specified"
            exit 1
        fi
        ;;
    *)
        break
        ;;
    esac
    shift
done

echo Namespace: ${namespace}
echo Verbose: ${verbose}
echo Metrics: ${metrics}
echo Regional Registry: ${regional}
echo International: ${international}
echo Cluster Region: ${clusterRegion}
echo All Regions: ${allRegions}
echo Hyperkube Image: ${hyperkubeImage}

if [[ -n ${registry} ]]; then
    echo Registry: ${registry}
    setRegistry="--set image.registry=${registry}"
fi

if [[ -n ${environment} ]]; then
    echo Registry Environment: ${environment}
    setEnvironment="--set image.name=armada_performance_${environment}/registry"
fi

if [[ -z ${registrykey} ]]; then
    registrykey=${PROD_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}
    if [[ -z ${registrykey} ]]; then
        echo "Registry api key not specified"
        exit 1
    fi
fi

if [[ -n ${registryDNS} ]]; then
    echo Registry DNS: ${registryDNS}

    # split the DNS (ip:host) into ip and host
    registryHost=$(echo ${registryDNS} | cut -d$":" -f2)
    registryIp=$(echo ${registryDNS} | cut -d$":" -f1)

    setHost="--set host=${registryHost}"
    setIp="--set ip=${registryIp}"
    echo ip: ${registryIp}
    echo host: ${registryHost}
fi

# Delete any existing job from previous runs
helm uninstall ${jobName} --namespace=${namespace} 2>/dev/null

# Create a new config map so that we can pass in the supplied parameters to the job(pod) creation
kubectl delete configmap perf-registry-config --ignore-not-found=true --namespace=${namespace}
kubectl create configmap perf-registry-config --namespace=${namespace} --from-literal=PERF_REG_HYPERKUBE=${hyperkubeImage} --from-literal=PERF_REG_CLUSTERREGION=${clusterRegion} --from-literal=PERF_REG_ALLREGIONS=${allRegions} --from-literal=PERF_REG_INTERNATIONAL=${international} --from-literal=PERF_REG_VERBOSE=${verbose} --from-literal=PERF_REG_REGIONAL=${regional} --from-literal=PERF_REG_METRICS=${metrics} --from-literal=PERF_REG_REGKEY=${registrykey}

# Use helm to install the chart and execute the tests as a kubernetes job on the cluster
helm install ${jobName} ../imageDeploy/registry --namespace=${namespace} --set metricsPrefix=${METRICS_PREFIX} --set metricsOS=${METRICS_OS} --set k8sVersion=${K8S_SERVER_VERSION} ${setRegistry} ${setEnvironment} ${setIp} ${setHost}
