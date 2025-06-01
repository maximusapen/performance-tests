#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

#Defaults
registry=""
environment=""
get_namespace="incluster-apiserver-target"
instances=10
replicas=5
throughput=20
duration=300
namespace=default

perf_dir=/performance
armada_perf_dir=${perf_dir}/armada-perf
helm_dir=/usr/local/bin

while test $# -gt 0; do
    case "$1" in
    -h | --help)
        echo "run_incluster_apiserver.sh - runs kubernetes incluster apiserver test"
        echo " "
        echo "run_incluster_apiserver.sh [options]"
        echo " "
        echo "options:"
        echo "-h, --help                        show brief help"
        echo "-e, --environment environment     Registry environment namespace (e.g dev7, stage1, etc.)"
        echo "-g, --registry registry_url       image registry location"
        echo "-n, --namespace k8s_namespace     kubernetes namespace for deployment"
        echo "-i, --instances                   Number of instances to run - each instance has its own deployment & Service Account"
        echo "-r, --replicas                    The number of pod replicas for each instance"
        echo "-t, --throughput                  The max throughput for each pod - so total throughput will be throughput x replicas x instances. 0 means do not limit throughput"
        echo "-m, --get_namespace               The namespace that will be used in the test for the get requests"
        exit 0
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
    -i | --instances)
        shift
        if test $# -gt 0; then
            instances=$1
        else
            echo "instances number not specified"
            exit 1
        fi
        shift
        ;;
    -r | --replicas)
        shift
        if test $# -gt 0; then
            replicas=$1
        else
            echo "replicas number not specified"
            exit 1
        fi
        shift
        ;;
    -t | --throughput)
        shift
        if test $# -gt 0; then
            throughput=$1
        else
            echo "throughput number not specified"
            exit 1
        fi
        shift
        ;;

    -m | --get_namespace)
        shift
        if test $# -gt 0; then
            get_namespace=$1
        else
            echo "get_namespace value not specified"
            exit 1
        fi
        shift
        ;;
    *)
        args="${args} "$*
        break
        ;;
    esac
done

if [[ -n ${registry} ]]; then
    echo Registry: ${registry}
    setRegistry="--set image.registry=${registry}"
fi

if [[ -n ${environment} ]]; then
    echo Registry Environment: ${environment}
    setEnvironment="--set image.name=armada_performance_${environment}/incluster-apiserver"
fi

echo "Instances: ${instances}, Replicas: ${replicas}, Throughput: ${throughput}, Get_Namespace: ${get_namespace}"

# Each instance will get from a different namespace
declare -a created_namespaces
for ((i = 1; i <= ${instances}; i = i + 1)); do
    ns="${get_namespace}${i}"
    namespaces=$(kubectl get namespaces | grep -w "${ns}" | wc -l)
    if [ "$namespaces" -eq 0 ]; then
        echo "Get_namespace  ${ns} does not exist, will create and populate with pods/secrets"
        # Ensure secrets exist 
        source ${armada_perf_dir}/automation/bin/setupRegistryAccess.sh ${ns}
        # Need to call this to ensure correct permissions given to the namespaces
        source ${armada_perf_dir}/automation/bin/enableRootUserOnOpenshift.sh ${ns} 
        helm install "incluster-apiserver-target" --namespace ${ns} ../imageDeploy/incluster-apiserver-target/ --set parameters.getNamespace=${ns}
        created_namespaces+=(${ns})
    fi
done

echo "Cleaning up any left over instances"
for c in $(helm list --namespace ${namespace} | grep incluster-apiserver | awk '{print $1}'); do
    echo "Deleting incluster-apiserver $c in namespace ${namespace}"
    helm uninstall $c --namespace ${namespace}
done

echo "Cleaning up any left over clusterroles/rolebindindings"
for c in $(kubectl get clusterrole | grep incluster-apiserver | awk '{print $1}'); do
    echo "Deleting clusterrole $c"
    kubectl delete clusterrole $c
done
for c in $(kubectl get clusterrolebinding | grep incluster-apiserver | awk '{print $1}'); do
    echo "Deleting clusterrolebinding $c"
    kubectl delete clusterrolebinding $c
done

echo "Installing ${instances} instances of incluster-apiserver"
for ((i = 1; i <= ${instances}; i = i + 1)); do
    helm install "incluster-apiserver-${i}" --namespace ${namespace} ../imageDeploy/incluster-apiserver/ --set parameters.runtime=${duration} --set replicaCount=${replicas} --set parameters.throughput=${throughput} --set parameters.getNamespace="${get_namespace}${i}" --set prefix=test${i} --set namespace=${namespace} ${setEnvironment} ${setRegistry}
done

echo "Test Running:"
kubectl get pods --namespace ${namespace} -l app=incluster-apiserver
echo "Waiting for ${duration} seconds for test to complete"
# Wait while the test runs - allow and extra 10 secs so the load pods will have finished
sleeptime=$((${duration} + 10))
sleep ${sleeptime}

total_runtime=0
total_throughput=0
total_requests=0
total_success=0
total_errors=0
overall_min_response=0
overall_max_response=0
total_mean_response=0
number_pods=0
# Get results
for pod in $(kubectl get pods --namespace ${namespace} --no-headers -l app=incluster-apiserver | awk '{print $1}'); do
    output=$(kubectl logs --namespace ${namespace} ${pod} | grep Summary | tail -n 1)
    echo "$output"
    if [ -z "$output" ]; then
        echo "WARNING - no Summary found in logs - dumping full logs"
        # If Pod hasn't started this may fail
        set +e
        kubectl logs --namespace ${namespace} ${pod}
        if [[ $? -ne 0 ]]; then
            kubectl describe pod --namespace ${namespace} ${pod} 
        fi
        set -e
        continue
    fi
    runtime=$(echo "${output}" | jq -r .RunTime)
    total_runtime=$((total_runtime + runtime))

    throughput=$(echo "${output}" | jq -r .Throughput)
    # Throughput is a float so need to use awk to add
    total_throughput=$(awk "BEGIN {printf \"%.2f\n\", ${total_throughput} + ${throughput}}")

    num_requests=$(echo "${output}" | jq -r .NumRequests)
    total_requests=$((total_requests + num_requests))

    num_success=$(echo "${output}" | jq -r .NumSuccess)
    total_success=$((total_success + num_success))

    num_errors=$(echo "${output}" | jq -r .NumErrors)
    total_errors=$((total_errors + num_errors))

    if [ "$num_errors" -gt 0 ]; then
        echo "ERROR occurred in Pod ${pod} - dumping logs"
        kubectl logs --namespace ${namespace} ${pod}
    fi

    min_response_time=$(echo "${output}" | jq -r .MinResponseTime)
    if [ "$overall_min_response" -eq 0 ] || [ "$min_response_time" -lt "$overall_min_response" ]; then
        overall_min_response=$min_response_time
    fi
    mean_response_time=$(echo "${output}" | jq -r .MeanResponseTime)
    total_mean_response=$((total_mean_response + mean_response_time))
    number_pods=$((number_pods + 1))

    max_response_time=$(echo "${output}" | jq -r .MaxResponseTime)
    if [ "$max_response_time" -gt "$overall_max_response" ]; then
        overall_max_response=$max_response_time
    fi
    items=$(echo "${output}" | jq -r .ItemsPerResponse)
done
overall_mean_response=$((total_mean_response / number_pods))

# Get Error% to 5 decimal places
overall_error_percent=$(awk "BEGIN {printf \"%.5f\n\", ${total_errors}/${total_requests}}")

echo "Total Requests: ${total_requests} , Total throughput: ${total_throughput}, Error Percent: ${overall_error_percent}, Min: ${overall_min_response}, Max: ${overall_max_response}, Mean: ${overall_mean_response}"

# Write data to Jmeter format so we can send it to metrics service
if [ -z $resCSVFile ]; then
    resCSVFile="incluster-apiserver-results.csv"
fi

echo "sampler_label,aggregate_report_count,average,aggregate_report_median,aggregate_report_90%_line,aggregate_report_95%_line,aggregate_report_99%_line,aggregate_report_min,aggregate_report_max,aggregate_report_error%,aggregate_report_rate,aggregate_report_bandwidth,aggregate_report_stddev" >${resCSVFile}
echo "TOTAL,${total_requests},${overall_mean_response},0,0,0,0,${overall_min_response},${overall_max_response},${overall_error_percent}%,${total_throughput},0,0" >>${resCSVFile}
