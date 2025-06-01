#!/bin/bash -e

# Install a Daemonset that will generate logs, and time how long it takes to gather the logs.
# Usage: run_logspeed.sh <num_repeats> <true|false>
#   <num_repeats>: How many times to repeat gathering the logs from all pods
#   <true|false> Whether or not to run the log requests in parallel
#       true: Run kubectl logs requests for all nodes in parallel
#       false: Run kubectl logs requests for all nodes sequentially

# The logs are generated in an Init Container - so when the pod goes Running we know they have finished generating the logs
waitForLogs() {
    maxWaitTime=600
    curWaitTime=0
    podsReady=false
    pollingInterval=60

    printf "\n%s - Checking all pods are Running.\n" "$(date +%T)"

    while [[ ${curWaitTime} -lt ${maxWaitTime} ]]; do
        pods=$(kubectl get pods -n ${namespace} -l name=logger --no-headers | grep " Running " | wc -l)
        nodes=$(kubectl get nodes --no-headers | wc -l)
        if ((${pods} == ${nodes})); then
            printf "\n%s - All pods are ready.\n" "$(date +%T)" 
            podsReady=true
            break
        else
            printf "%s - Waiting of for all pods (Expecting %d) to be running. Running pods: %d\n" "$(date +%T)"  "${nodes}" "${pods}"
            sleep ${pollingInterval}
            ((curWaitTime += ${pollingInterval}))
        fi
    done

    if [[ "${podsReady}" != true ]]; then
        printf "\n%s - Gave up waiting for the to be Running Exiting.\n\n" "$(date +%T)"
        exit 1
    fi
}

# KUBECONFIG environment must be set
if [[ -z "${KUBECONFIG}" ]]; then
    printf "KUBECONFIG not set. Exiting.\n"
    exit 1
fi

# How many repeats of the test to run.
num_repeats=$1
if [[ -z "${num_repeats}" ]]; then
    num_repeats=1
else
    concurrent=$2
    if [[ -z "${concurrent}" ]]; then
        concurrent="false"
    fi    
fi

printf "\n%s - Running %d repeats of gathering logs, concurrent = %s \n" "$(date +%T)" "${num_repeats}" "${concurrent}"
namespace="default"

OIFS=$IFS
IFS=$'\n'

# Ensure old runs are tidied up first
kubectl delete -f ./log-gen.yaml -n ${namespace} --ignore-not-found

kubectl apply -f ./log-gen.yaml -n ${namespace}

waitForLogs

total_log_gets=0
total_seconds=0
total_log_bytes=0
max_seconds=0
min_seconds=999999
total_failures=0
# This controls how many concurrent log gets per pod will be run - so if set to 5 and it is a 5 node cluster
# there will be a total of 25 concurrent requests.
concurrency=1
if [[ "${concurrent}" = "true" ]]; then
    for (( j=1; j<=${num_repeats}; j++ )); do
        SECONDS=0
        count=0
        failures=0
        for (( k=1; k<=${concurrency}; k++ )); do
            for i in `kubectl get pods -n ${namespace} -l name=logger --no-headers -o=wide`; do
                podName=$(echo ${i}| awk '{print $1}')
                nodeName=$(echo ${i}| awk '{print $7}')
                # Time how long it takes to get the logs
                kubectl logs -n ${namespace} -c genlogs ${podName} > ${podName}-${k}.log &
            done
        done
        printf "\n%s - Waiting for all logs to be gathered \n" "$(date +%T)"
        wait
        log_get_time=$SECONDS
        # If logs get trimmed this can be quick, so have a minimm of 1 sec
        if [[ ${log_get_time} -lt 1 ]]; then
            log_get_time=1
        fi
        repeat_log_bytes=0
        for (( k=1; k<=${concurrency}; k++ )); do
            for i in `kubectl get pods -n ${namespace} -l name=logger --no-headers -o=wide`; do
                podName=$(echo ${i}| awk '{print $1}')
                nodeName=$(echo ${i}| awk '{print $7}')
                log_size=$(stat -c%s ${podName}-${k}.log)
                # If log size is 0 count as a failure
                if [[ ${log_size} -eq 0 ]]; then
                    failures=$((failures + 1))
                fi 
                repeat_log_bytes=$(($repeat_log_bytes + $log_size))
                single_bytes_per_second=$(($log_size/$log_get_time))
                printf "\n%s - Got concurrent logs from pod %s on node %s in %s seconds. File size was %d . Log download rate was %d Bytes per second" "$(date +%T)" "${podName}-${k}" "${nodeName}" "${log_get_time}" "${log_size}" "${single_bytes_per_second}"
                rm ${podName}-${k}.log
            done
        done
        bytes_per_second=$((${repeat_log_bytes}/${log_get_time}))
        printf "\n%s - Got logs from all pods in %s seconds. Total File size was %d . Log download rate was %d Bytes per second. Failures: %d " "$(date +%T)" "${log_get_time}" "${repeat_log_bytes}" "${bytes_per_second}" "${failures}"

        total_failures=$(($total_failures + $failures))
        total_seconds=$(($total_seconds + $log_get_time))
        total_log_gets=$((total_log_gets + 1))
        total_log_bytes=$(($total_log_bytes + $repeat_log_bytes))
        if [[ ${log_get_time} -lt ${min_seconds} ]]; then
            min_seconds=${log_get_time}
        fi
        if [[ ${log_get_time} -gt ${max_seconds} ]]; then
            max_seconds=${log_get_time}
        fi
    done
else
    failures=0
    for (( j=1; j<=${num_repeats}; j++ )); do
        for i in `kubectl get pods -n ${namespace} -l name=logger --no-headers -o=wide`; do
            podName=$(echo ${i}| awk '{print $1}')
            nodeName=$(echo ${i}| awk '{print $7}')
            # Time how long it takes to get the logs
            SECONDS=0
            kubectl logs -n ${namespace} -c genlogs ${podName} > ${podName}.log
            log_get_time=$SECONDS
        
            # If logs get trimmed this can be quick, so have a minimm of 1 sec
            if [[ ${log_get_time} -lt 1 ]]; then
                log_get_time=1
            fi
            log_size=$(stat -c%s ${podName}.log)
            # If log size is 0 count as a failure
            if [[ ${log_size} -eq 0 ]]; then
                    failures=$((failures + 1))
            fi
            bytes_per_second=$((${log_size}/${log_get_time}))
            printf "\n%s - Got logs from pod %s on node %s in %s seconds. File size was %d . Log download rate was %d Bytes per second" "$(date +%T)" "${podName}" "${nodeName}" "${log_get_time}" "${log_size}" "${bytes_per_second}"
            
            total_failures=$(($total_failures + $failures))
            total_seconds=$(($total_seconds + $log_get_time))
            total_log_gets=$((total_log_gets + 1))
            total_log_bytes=$(($total_log_bytes + $log_size))
            if [[ ${log_get_time} -lt ${min_seconds} ]]; then
                min_seconds=${log_get_time}
            fi
            if [[ ${log_get_time} -gt ${max_seconds} ]]; then
                max_seconds=${log_get_time}
            fi
            rm ${podName}.log

        done
    done


fi
mean_seconds=$((${total_seconds}/${total_log_gets}))
overall_rate=$((${total_log_bytes}/${total_seconds}))

kubectl delete -f ./log-gen.yaml -n ${namespace}

printf "\n%s - Test completed - Min time: %s, Mean time: %s, Max time: %s, Overall transfer rate: %s bytes per second, Failures: %s \n\n" "$(date +%T)" "${min_seconds}" "${mean_seconds}" "${max_seconds}" "${overall_rate}" "${total_failures}"
