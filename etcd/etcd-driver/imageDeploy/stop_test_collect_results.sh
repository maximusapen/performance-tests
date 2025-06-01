#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020, 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to stop etcd-driver test and collect results
# Prereqs - Kubectl with an appropriate KUBECONFIG set, and helm must be already configured.

if [[ $# -lt 2 ]]; then
    echo "Usage: `basename $0` <prefix> <namespace> [<helm chart> [<true>]]"
    echo "<etcd cluster name> = Must match etcd-operator instance (i.e. can only test on one cluster at a time)."
    echo "<namespace> = The namespace to create etcd-driver chart in"
    echo "[<helm chart>] = The helm chart (defaults to etcd-driver)"
    echo "[<true>] = Skip test end trigger"
    exit 1
fi

. etcd-perftest-config

prefix=$1
namespace=$2

helm_chart=etcd-driver
if [[ $# -ge 3 ]]; then
    helm_chart=$3
fi

skip_trigger=$4

#. etcd-perftest-config

dt=$(date +"%Y-%m-%d-%H-%M")

mkdir -p backup

if [[ -z ${skip_trigger} ]]; then
    if [[ ${USE_CERTIFICATES} == "true" ]]; then
        pod=$(kubectl get pods -n ${namespace} --no-headers | grep ${prefix}-${helm_chart} | head -1 | awk '{print $1}')
        container=$(kubectl -n ${namespace} get pod ${pod} -o=jsonpath='{.spec.containers[*].name}' | head -1)
        if [[ -z ${pod} && -z ${container} ]]; then
            echo "ERROR: Couldn't find test pod/container [pod=${pod} container=${container}]. Couldn't terminate test"
            exit 1
        fi
        date
        certs="--cert=/etc/etcdtls/operator/etcd-tls/etcd-client.crt --key=/etc/etcdtls/operator/etcd-tls/etcd-client.key --cacert=/etc/etcdtls/operator/etcd-tls/etcd-client-ca.crt"
        kubectl -n ${namespace} exec ${pod} -c ${container} -- sh -c "ETCDCTL_API=3 /etcdctl ${certs} --endpoints=${ETCD_VIP_ENDPOINTS} put /test1/end true"
        if [[ $? -ne 0 ]]; then
            echo "ERROR: Failed to write test end key"
            exit 1
        fi
        sleep 30
        kubectl -n ${NAMESPACE} exec ${pod} -c ${container} -- sh -c "ETCDCTL_API=3 /etcdctl ${certs} --endpoints=${ETCD_VIP_ENDPOINTS} put /test1/end true"
    else
        # Find LB
        endpoints=$(kubectl -n ${namespace} get svc ${prefix}-client-service-lb -o=jsonpath='{.status.loadBalancer.ingress[*].ip}{":"}{.spec.ports[*].nodePort}{"\n"}')
        echo "Endpoints: ${endpoints}"

        if [[ -z ${endpoints} ]]; then
            echo "ERROR: Service ${prefix}-client-service-lb couldn't be found. Couldn't terminate test"
            exit 1
        fi

        # Trigger test termination
        date
        etcdctl put --endpoints=${endpoints} /test1/end true
        if [[ $? -ne 0 ]]; then
            echo "ERROR: Failed to write test end key"
            exit 1
        fi
        # Send a second time, since seen cases where notifications didn't get sent to multiple containers
        # Basically I think the watches are overloaded and the updates need to stop before the test end watch reacts
        sleep 30
        etcdctl put --endpoints=${endpoints} /test1/end true
    fi
    echo "Stop: log label: ${dt}"

    # Wait a bit
    sleep 30
    date
else
    echo "Skipping test termination"
fi

# Collect results
kubectl -n ${namespace} get pvc --no-headers | grep ${prefix}-${helm_chart} #1> /dev/null 2>&1
haspvc=$?
echo haspvc $haspvc

first=true
if [[ ${haspvc} -ne 0 ]]; then
    echo "Pull results from pods"
    for pod in `kubectl get pods -n ${namespace} --no-headers | grep ${prefix}-${helm_chart} | awk '{print $1}' `; do
        for container in `kubectl -n ${namespace} get pod ${pod} -o=jsonpath='{.spec.containers[*].name}'`; do
            if [[ ${container} != *"-lease"* ]]; then
                kubectl -n ${namespace} cp ${pod}:/churn_results.csv churn_results.${pod}.${container}.${dt}.csv -c ${container}
                if [[ ${first} == "true" ]]; then
                    head -1 churn_results.${pod}.${container}.${dt}.csv >> churn_results.${dt}.csv
                    first=false
                fi
                grep pattern-summary churn_results.${pod}.${container}.${dt}.csv >> churn_results.${dt}.csv
                mv churn_results.${pod}.${container}.${dt}.csv backup
            fi
        done
    done
else
    echo "Pull results from pvc"
    pod=$(kubectl get pods -n ${namespace} --no-headers | grep ${prefix}-${helm_chart} | head -1 | awk '{print $1}')
    container=$(kubectl -n ${namespace} get pod ${pod} -o=jsonpath='{.spec.containers[*].name}' | head -1)
    mkdir -p backup/${dt}
    pushd backup/${dt}
    kubectl -n ${namespace} exec ${pod} -c ${container} -- tar czf - /results | tar xf -
    popd
    for i in `find backup/${dt} -name churn_results.csv`; do
        echo "Found " $i
        if [[ ${first} == "true" ]]; then
            head -1 ${i} > churn_results.${dt}.csv
            first=false
        fi
        grep pattern-summary ${i} >> churn_results.${dt}.csv
    done
fi

cnt=$(grep pattern-summary churn_results.*${dt}.csv | wc -l)
if [[ -z ${cnt} ]]; then
    cnt="No"
fi
echo "${cnt} pod/containers reported summary statistics"

# Grab the logs
for pod in `kubectl get pods -n ${namespace} --no-headers | grep ${prefix}-${helm_chart} | awk '{print $1}' `; do
    for container in `kubectl -n armada get pod ${pod} -o=jsonpath='{.spec.containers[*].name}'`; do
        kubectl -n ${namespace} logs ${pod} -c ${container} > backup/logs.${pod}.${container}.${dt}.log
    done
done

./get_etcd_logs.sh

./summarize_errors.sh ${dt}
start_time=$(sort backup/${dt}.test.dates | head -1)
end_time=$(sort backup/${dt}.test.dates | tail -1)

etcd_dt=$(cat etcd_logs_label.txt)

# Lines from log file won't be collected unles there is a match with $start_time. Matching against a ten minute window
# greatly increases the chance of a match.
start_time="${start_time%?}*"
echo "Etcd log analysis parameters: ${etcd_dt} ${start_time} ${end_time}"
./summarize_etcd_errors.sh ${etcd_dt} "${start_time}" "${end_time}"
