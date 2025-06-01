#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2018, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

#Defaults
jobfile=""
registry=""
environment=""
namespace="default"
storagemode="ReadWriteMany"
storagesize="20Gi"
numjobs="1"
storageclass="ibmc-block-bronze"
scheduler="default-scheduler"
metrics=false
verbose=false

jobName=persistent-storage

while test $# -gt 0; do
    case "$1" in
    -h | --help)
        echo "perf-persistent-storage - runs persistent storage performance tests"
        echo " "
        echo "perf-persistent-storage [options]"
        echo " "
        echo "options:"
        echo "-h, --help                  	show brief help"
        echo "-t, --testname testname   	Name of test in Jenkins - only used if sending alerts to RazeeDash"
        echo "-e, --environment environment     Registry environment namespace (e.g stage1, stage2, etc.)"
        echo "-g, --registry registry_url	image registry location"
        echo "-n, --namespace k8s_namespace	kubernetes namespace for deployment"
        echo "-m, --metrics			request results are sent to IBM Cloud monitoring service"
        echo "-v, --verbose			log test results to stdout"
        echo "-j, --jobfile fiojobfile	filepath of fio job file"
        echo "-s, --size size			persistent storage size (e.g. 20Gi)"
        echo "-p, --numjobs numjobs		Number of concurrent fio jobs"
        echo "-c, --class storage_class	persistent storage class name (e.g. ibmc-block-bronze)"
        echo "-x, --scheduler scheduler_name    pod scheduler"
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
    -s | --size)
        shift
        if test $# -gt 0; then
            storagesize=$1
        else
            echo "Storage size not specified"
            exit 1
        fi
        shift
        ;;
    -p | --numjobs)
        shift
        if test $# -gt 0; then
            numjobs=$1
        else
            echo "numjobs not specified, using default of $numjobs"
        fi
        shift
        ;;
    -c | --class)
        shift
        if test $# -gt 0; then
            storageclass=$1
        else
            echo "storage class name not specified"
            exit 1
        fi
        shift
        ;;
    -x | --scheduler)
        shift
        if test $# -gt 0; then
            scheduler=$1
        else
            echo "pod scheduler name not specified"
            exit 1
        fi
        shift
        ;;
    *)
        break
        ;;
    esac
done

if [[ $storageclass == *"block"* ]] || [[ $storageclass == *"rbd"* ]]; then
    storagemode="ReadWriteOnce"
fi

echo Namespace: ${namespace}
echo Verbose: ${verbose}
echo Job file: ${jobfile}
echo Pod scheduler: ${scheduler}
echo Storage Class Name: ${storageclass}
echo Storage Size: ${storagesize}
echo FIO numjobs: ${numjobs}
echo Metrics: ${metrics}

if [[ -n ${registry} ]]; then
    echo Registry: ${registry}
    setRegistry="--set image.registry=${registry}"
fi

if [[ -n ${environment} ]]; then
    echo Registry Environment: ${environment}
    setEnvironment="--set image.name=armada_performance_${environment}/persistent-storage"
fi

perfdir="/var/perfps/${storageclass}-${storagesize}"

# Delete any existing job from previous runs
helm uninstall ${jobName} --namespace=${namespace} 2>/dev/null

# Create a new config map so that we can pass in the supplied parameters to the job(pod) creation
kubectl delete configmap perf-persistent-storage-config --ignore-not-found=true --namespace=${namespace}
kubectl create configmap perf-persistent-storage-config --namespace=${namespace} --from-literal=PERF_PS_VERBOSE=${verbose} --from-literal=PERF_PS_METRICS=${metrics} --from-literal=PERF_PS_TESTNAME=${testname} --from-literal=PERF_PS_DIR="${perfdir}" --from-literal=PERF_PS_JOBFILE="${jobfile}" --from-literal=PERF_PS_NUMJOBS="${numjobs}"

# The pvc can take a while to actually delete - so we need to check it has gone before we try install again,
# otherwise the install will fail.
counter=0
attempts=60
while [ $counter -lt $attempts ]; do
    pending_resources="$(kubectl get pvc --namespace ${namespace} perf-pvc 2>/dev/null)"
    if [ -n "${pending_resources}" ]; then
        echo "perf-pvc still exists. ${counter}/${attempts} tests completed; retrying."
        echo "${pending_resources}" 1>&2
        ((++counter))
        sleep 10
    else
        break
    fi
done

if [ $counter -eq $attempts ]; then
    echo "Persistent Volume Claim perf-pvc failed to delete in time. Exiting"
    kubectl describe pvc --namespace=${namespace} perf-pvc
    exit 1
fi

# Use helm to install the chart and execute the tests as a kubernetes job on the cluster
helm install ${jobName} ../imageDeploy/persistent-storage --namespace=${namespace} --set metricsPrefix=${METRICS_PREFIX} --set metricsOS=${METRICS_OS} --set k8sVersion=${K8S_SERVER_VERSION} --set pvc.accessMode=${storagemode} --set pvc.storageSize=${storagesize} --set pvc.storageClassName=${storageclass} --set pod.scheduler=${scheduler} ${setRegistry} ${setEnvironment}
