#!/bin/bash -e
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
namespace="default"
concurrency=1
duration=120
client_only=false
load_balancer=false
netperf=false
startpod=1

perf_dir=/performance
armada_perf_dir=${perf_dir}/armada-perf
helm_dir=/usr/local/bin
helm_config_dir=${armada_perf_dir}/helm/config

while test $# -gt 0; do
    case "$1" in
    -h | --help)
        echo "run_iperf.sh - runs kubernetes based Linux iperf server & client"
        echo " "
        echo "run_iperf.sh [options]"
        echo " "
        echo "options:"
        echo "-h, --help                        show brief help"
        echo "-e, --environment environment     Registry environment namespace (e.g dev7, stage1, etc.)"
        echo "-g, --registry registry_url       image registry location"
        echo "-n, --namespace k8s_namespace     kubernetes namespace for deployment"
        echo "-c, --concurrency                 the number of concurrent servers & clients to run"
        echo "-o, --client-only                 skip the iperf server setup, and just run the client (Server must have been configured by aprevious run)"
        echo "-l, --loadbalancer                use the load balancer"
        echo "-u, --usenetperf                  use the preconfigured netperf clients"
        echo "-s, --startpod                    make this the first pod to use in test. Must be >= 1"
        exit 0
        ;;
    -c | --concurrency)
        shift
        if test $# -gt 0; then
            concurrency=$1
        else
            echo "Concurrency not specified"
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
    -o | --client-only)
        client_only=true
        shift
        ;;
    -l | --loadbalancer)
        load_balancer=true
        shift
        ;;
    -u | --usenetperf)
        netperf=true
        shift
        ;;

    -s | --startpod)
        shift
        if test $# -gt 0; then
            startpod=$1
        else
            echo "startpod number not specified"
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

if [[ -z ${CLIENT_KUBECONFIG} ]]; then
    if [[ -z ${KUBECONFIG} ]]; then
        echo "Either CLIENT_KUBECONFIG or KUBECONFIG must be set"
        exit 1
    else
        CLIENT_KUBECONFIG=${KUBECONFIG}
    fi
fi
if [[ -z ${SERVER_KUBECONFIG} ]]; then
    if [[ -z ${KUBECONFIG} ]]; then
        echo "Either SERVER_KUBECONFIG or KUBECONFIG must be set"
        exit 1
    else
        SERVER_KUBECONFIG=${KUBECONFIG}
    fi
fi

echo "Using ${CLIENT_KUBECONFIG} for iperf client cluster"
echo "Using ${SERVER_KUBECONFIG} for iperf server cluster"
echo "Load balancer in use? $load_balancer"
lb_args=""

if [ "$client_only" = false ]; then
    # Setup the server
    export KUBECONFIG=${SERVER_KUBECONFIG}

    # Setup registry

    ${armada_perf_dir}/automation/bin/setupRegistryAccess.sh ${namespace} false

    for ((i = ${startpod}; i <= $((concurrency + startpod - 1)); i = i + 1)); do
        echo ${i}
        ./iperfserver.sh --registry ${registry} --namespace ${namespace} --environment ${environment} --pods 1 --id ${i} &
    done
    jobs
    wait

    # Give some time for the pods to schedule
    sleep 60

else
    echo "--client-only is set, skipping server setup"
fi

tstamp=$(date +"%Y%m%d_%H%M%S")

if [ "$netperf" = false ]; then
    # Setup registry
    export KUBECONFIG=${CLIENT_KUBECONFIG}

    ${armada_perf_dir}/automation/bin/setupRegistryAccess.sh ${namespace} false
fi

for ((i = ${startpod}; i <= $((concurrency + startpod - 1)); i = i + 1)); do
    # Get the port for the server
    export KUBECONFIG=${SERVER_KUBECONFIG}
    if [ "$load_balancer" = true ]; then
        lb_args="--loadbalancer"
        ip=$(kubectl get svc iperfserver-lb-service-$i -n ${namespace} -o=jsonpath='{.status.loadBalancer.ingress[0].ip}')
    else
        node=$(kubectl get pod -l "app=iperfserver-$i" -n ${namespace} -o=jsonpath='{.items[0].spec.nodeName}')
        ip=$(kubectl get node ${node} -o jsonpath='{ $.status.addresses[?(@.type=="ExternalIP")].address }')
    fi

    export KUBECONFIG=${CLIENT_KUBECONFIG}

    if [ "$netperf" = false ]; then
        ./iperfclient.sh --registry ${registry} --namespace ${namespace} --environment ${environment} --id ${i} ${lb_args} --address ${ip} -t $duration -J ${args} &
    else
        ./netperfclient.sh --namespace ${namespace} --id ${i} --address ${ip} --prefix ${tstamp} ${lb_args} -t $duration -J ${args} &
    fi
done

jobs
wait
# Wait for the duration specified plus 1 minute before gathering results
let "wait_time = ${duration} + 60"
sleep $wait_time

export KUBECONFIG=${SERVER_KUBECONFIG}
echo "Pod information for the iperf server:"
kubectl get pods -n ${namespace} -o=wide

export KUBECONFIG=${CLIENT_KUBECONFIG}
echo "Pod information for the iperf client:"
kubectl get pods -n ${namespace} -o=wide

mkdir -p results
MbitPSec=0

IFS=$'\n'

if [ "$netperf" = false ]; then
    for pods in $(kubectl get pods -n ${namespace} --no-headers | grep client); do
        # Skip clients that may have been left from previous tests
        postfix=${pods#iperfclient-}
        if [[ ${postfix%%-*} -le $((concurrency + startpod - 1)) ]]; then
            podName=$(echo $pods | cut -d$' ' -f1)
            echo "======== Detailed result from pod $podName are in results/${tstamp}_$podName ========"
            kubectl logs $podName -n ${namespace} >results/${tstamp}_$podName
        fi
    done
fi

for results in $(ls results/${tstamp}*); do
    podBitsPerSecond=$(grep -v "^args " $results | jq '.end.sum_received.bits_per_second')
    podBitsPerSecond=${podBitsPerSecond%.*}
    MbitPSec=$((podBitsPerSecond / 1000000))
    TotalMbitPSec=$((TotalMbitPSec + MbitPSec))
    echo "Bandwidth for pod ${results#results/${tstamp}_}: $MbitPSec Mbits/second"
done

echo "Total throughput: $TotalMbitPSec Mbits/second"
