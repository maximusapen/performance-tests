#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

#Defaults
jobfile=""
registry=""
environment=""
namespace="default"
metrics=false
verbose=false

jobName=local-storage

while test $# -gt 0; do
    case "$1" in
    -h | --help)
        echo "perf-local-storage - runs local storage performance tests"
        echo " "
        echo "perf-local-storage [options]"
        echo " "
        echo "options:"
        echo "-h, --help                  	show brief help"
        echo "-t, --testname testname   	Name of test in Jenkins - only used if sending alerts to RazeeDash"
        echo "-e, --environment environment     Registry environment namespace (e.g stage1, stage2, etc.)"
        echo "-g, --registry registry_url	image registry location"
        echo "-n, --namespace k8s_namespace	kubernetes namespace for deployment"
        echo "-m, --metrics			request results are sent to metrics service"
        echo "-v, --verbose			log test results to stdout"
        echo "-j, --jobfile fiojobfile	filepath of fio job file"
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
    -j | --jobfile)
        shift
        if test $# -gt 0; then
            jobfile=$1
        else
            echo "Job file not specified"
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
    -t | --testname)
        shift
        if test $# -gt 0; then
            testname=$1
        else
            echo "Test name not specified - no alerts will be sent to RazeeDash"
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
echo Job file: ${jobfile}
echo Metrics: ${metrics}

if [[ -n ${registry} ]]; then
    echo Registry: ${registry}
    setRegistry="--set image.registry=${registry}"
fi

if [[ -n ${environment} ]]; then
    echo Registry Environment: ${environment}
    setEnvironment="--set image.name=armada_performance_${environment}/persistent-storage"
fi

perfdir="/performance/local"

# Delete any existing job from previous runs
helm uninstall ${jobName} --namespace=${namespace} 2>/dev/null

# Create a new config map so that we can pass in the supplied parameters to the job(pod) creation
kubectl delete configmap perf-local-storage-config --ignore-not-found=true --namespace=${namespace}
kubectl create configmap perf-local-storage-config --namespace=${namespace} --from-literal=PERF_PS_VERBOSE=${verbose} --from-literal=PERF_PS_METRICS=${metrics} --from-literal=PERF_PS_TESTNAME=${testname} --from-literal=PERF_PS_DIR="${perfdir}" --from-literal=PERF_PS_JOBFILE="${jobfile}"

# Use helm to install the chart and execute the tests as a kubernetes job on the cluster
helm install ${jobName} ../imageDeploy/local-storage --namespace=${namespace} --set metricsPrefix=${METRICS_PREFIX} --set metricsOS=${METRICS_OS} --set k8sVersion=${K8S_SERVER_VERSION} ${setRegistry} ${setEnvironment}
