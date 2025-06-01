#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2018, 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

#Defaults
namespace="default"
verbose=false
metrics=false
testName=""

jobName=perf-sysbench

perf_dir=/performance

cluster="$(kubectl config current-context | cut -d '/' -f1)"

while test $# -gt 0; do
    case "$1" in
    -h | --help)
        echo "perf-sysbench - runs sysbench benchmark tests using daemonset"
        echo " "
        echo "perf-sysbench [options]"
        echo " "
        echo "options:"
        echo "-h, --help                	show brief help"
        echo "-g, --registry registry_url 	image registry location"
        echo "-t, --testname testname   	Name of test in Jenkins - only used if sending alerts to RazeeDash"
        echo "-e, --environment environment     registry environment namespace (e.g dev7, stage1, etc.)"
        echo "-n, --namespace k8s_namespace     kubernetes namespace for deployment"
        echo "-m, --metrics             	send results to metrics service"
        echo "-v, --verbose             	log test results to stdout"
        exit 0
        ;;
    -v | --verbose)
        verbose=true
        shift
        ;;
    -m | --metrics)
        metrics=true
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
    -t | --testname)
        shift
        if test $# -gt 0; then
            testname=$1
        else
            echo "Test name not specified - no alerts will be sent to RazeeDash"
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
    *)
        break
        ;;
    esac
done

echo Namespace: ${namespace}
echo Verbose: ${verbose}
echo Metrics: ${metrics}

if [[ -n ${registry} ]]; then
    echo Registry: ${registry}
    setRegistry="--set image.registry=${registry}"
fi

if [[ -n ${environment} ]]; then
    echo Registry Environment: ${environment}
    setEnvironment="--set image.name=armada_performance_${environment}/run-sysbench"
fi

# Delete any existing job from previous runs
helm uninstall ${jobName} --namespace=${namespace} 2>/dev/null
kubectl get pods -n sysbench
kubectl delete --force daemonset perf-sysbench-daemonset -n ${namespace} --ignore-not-found=true

# sorry this is horrid:
# - Get the cluster type
# - get a list of cluster workers details
# - Use jq to get the bits wanted (the node IP ,the machine flavor and id) and join them with = as they are going to be send as env's in a configmap
# - Remove the .'s and replace with _' and add 'node' to front as you can't have a number at the start of an env variable.
# - Add on a worker number formed using "-w"NR to give a worker of the format w<line number>. (Needed for use in Grafana.)
# - Send the list to a file called machineTypeList.
# - This list is passed through as an env-file to the pod. In the pod it is split into environemnt and worker number to be used in the metrics.

# Check the .provider in the json to see if this is a vpc or vpc-classic cluster
clusterType=$(${perf_dir}/bin/armada-perf-client2 cluster get --cluster ${cluster} --json | jq -r .provider)
echo Cluster Type: ${clusterType}

# If VPC* cluster
if [[ $clusterType == "vpc"* ]] || [[ $clusterType == "satellite"* ]]; then
    ${perf_dir}/bin/armada-perf-client2 worker ls --cluster ${cluster} --json | jq 'map( {id: .id, flavor: .flavor, ipAddress: .networkInterfaces[].ipAddress})' | jq -r '.[] | [.ipAddress, .flavor, .id] | join("=")' | sed 's/\./_/g' | sed 's/=/!/2' | sed 's/^/node/' | awk '{print $0"-w"NR}' | sed 's/\(!\).*\(-w\)/\2/g' >./machineTypeList
else
    # else its a classic cluster
    ${perf_dir}/bin/armada-perf-client2 worker ls --cluster ${cluster} --json | jq 'map( {id: .id, flavor: .flavor, ipAddress: .networkInformation.privateIP})' | jq -r '.[] | [.ipAddress, .flavor, .id] | join("=")' | sed 's/\./_/g' | sed 's/=/!/2' | sed 's/^/node/' | awk '{print $0"-w"NR}' | sed 's/\(!\).*\(-w\)/\2/g' >./machineTypeList
fi

cat ./machineTypeList

# Create a new config map so that we can pass in the supplied parameters to the job(pod) creation
kubectl delete configmap perf-sysbench-config --ignore-not-found=true --namespace=${namespace}
kubectl delete configmap perf-sysbench-machine-config --ignore-not-found=true --namespace=${namespace}
kubectl create configmap perf-sysbench-config --namespace=${namespace} --from-literal=PERF_SB_VERBOSE=${verbose} --from-literal=PERF_SB_METRICS=${metrics} --from-literal=PERF_SB_TESTNAME=${testname} --from-file=./machineTypeList
# create a configmap from the environment variable created earlier
kubectl create configmap perf-sysbench-machine-config --namespace=${namespace} --from-env-file=./machineTypeList

# Use helm to install the chart and execute the tests as a kubernetes job on the cluster
helm install ${jobName} ../imageDeploy/run-sysbench --namespace=${namespace} --set metricsPrefix=${METRICS_PREFIX} --set metricsOS=${METRICS_OS} --set k8sVersion=${K8S_SERVER_VERSION} ${setRegistry} ${setEnvironment}
