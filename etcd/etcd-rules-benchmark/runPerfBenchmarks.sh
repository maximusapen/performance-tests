#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# Script to run a selecton of etcd-rules-benchmark tests with various parameters
# It will also send the results to the metrics service

waitForTestComplete() {
    maxWaitTime=$1
    curWaitTime=0
    testComplete=false
    pollingInterval=60
    resultsFile=etcd-rules-benchmark-results.txt
    perftest="etcd-rules-benchmark"
    totalCpu=0
    totalMem=0
    numTopData=0
    while [[ ${curWaitTime} -lt ${maxWaitTime} ]]; do
        testPod=$(kubectl get pods -n armada | awk '{print $1}' | grep etcd-rules-benchmark)
        if [[ -z "${testPod}" ]]; then
            printf "%s - Waiting for etcd-rules-benchmark pod to start .\n" "$(date +%T)" 
            sleep ${pollingInterval}
            ((curWaitTime += ${pollingInterval}))
        else
            testResult=$(kubectl logs -n armada ${testPod} | grep "Benchmark test complete" )

            if [[ -z ${testResult} ]]; then
                lastStruct=$(kubectl logs -n armada ${testPod} --tail=1 | jq -r  .Struct)
                printf "%s - Waiting for Benchmark test complete message, last struct processed was ${lastStruct}, printing kubectl top data .\n" "$(date +%T)" 
                
                # Gather data around CPU/Memory used by the benchmark pod
                topData=$(kubectl top pods -n armada | grep etcd-rules-benchmark)
                echo ${topData}
                cpu=$(echo ${topData} | awk '{print $2}' | tr -d "m")
                memory=$(echo ${topData} | awk '{print $3}' | tr -d "Mi")
                ((totalCpu += ${cpu}))
                ((totalMem += ${memory}))
                ((numTopData++))

                sleep ${pollingInterval}
                ((curWaitTime += ${pollingInterval}))
            else
                testComplete=true
                printf "\n%s - Test complete, result: \n" "$(date +%T)"
                echo ${testResult}
                echo ${testResult} >> ${resultsFile}

                # Extract final metrics from the results
                aveTime=$(echo ${testResult} | jq -r '."Average time spent (ms)"')
                minTime=$(echo ${testResult} | jq -r '."Min time spent (ms)"')
                maxTime=$(echo ${testResult} | jq -r '."Max time spent (ms)"')
                numKeys=$(echo ${testResult} | jq -r '."Keys-Set"')
                concurrency=$(echo ${testResult} | jq -r '."RE Concurrency"' | tr -d "-")
                
                if [[ ${numTopData} -eq 0 ]]; then
                    aveCpu=0
                    aveMem=0
                else
                    aveCpu=$((totalCpu / numTopData))
                    aveMem=$((totalMem / numTopData))
                fi

                printf "\n%s - Sending Results, ave: %s , min: %s, max: %s , AveCpu: %s, AveMem: %s \n" "$(date +%T)" "${aveTime}" "${minTime}" "${maxTime}" "${aveCpu}" "${aveMem}"

                # Send results to metrics service
                metricsTestNameBase=${perftest}."keys${numKeys}"."concurrency${concurrency}"
                /performance/bin/send-to-bm -verbose -testname "${perftest}" -bmval "${aveTime}" -metricsTestName "${metricsTestNameBase}.ave"
                /performance/bin/send-to-bm -verbose -testname "${perftest}" -bmval "${minTime}" -metricsTestName "${metricsTestNameBase}.min"
                /performance/bin/send-to-bm -verbose -testname "${perftest}" -bmval "${maxTime}" -metricsTestName "${metricsTestNameBase}.max"
                /performance/bin/send-to-bm -verbose -testname "${perftest}" -bmval "${aveCpu}" -metricsTestName "${metricsTestNameBase}.cpu.ave"
                /performance/bin/send-to-bm -verbose -testname "${perftest}" -bmval "${aveMem}" -metricsTestName "${metricsTestNameBase}.memory.ave"
                break
            fi
        fi
    done

    if [[ "${testComplete}" != true ]]; then
        printf "\n%s - Gave up waiting for etcd-rules-benchmark test to complete \n" "$(date +%T)"
        exit 1
    fi
}

runBenchmark() {
    oneStructVar=$1
    multiStructVar=$2
    concurrencyVar=$3
    concurrencyNum=$4
    numberKeysVar=$5
    
    echo "Running test with oneStruct: ${oneStructVar}, multiStruct: ${multiStructVar}, concurrency: ${concurrencyVar}, concurrencyNum: ${concurrencyNum}, numberKeys: ${numberKeysVar}"
    # Install the application
    helm install etcd-rules-benchmark ${PERF_DIR}/armada-perf/etcd/etcd-rules-benchmark/performance/imageDeploy/etcd-rules-benchmark --namespace armada  --set oneStruct="${oneStructVar}" --set multipleStruct="${multiStructVar}" --set concurrency="${concurrencyVar}" --set concurrencyNum="${concurrencyNum}" --set numberKeys="${numberKeysVar}" --set secretName="${SECRET_NAME}"
    sleep 120
    # Wait for it to complete
    waitForTestComplete 5400

    helm uninstall etcd-rules-benchmark --namespace armada
    sleep 30
}

PERF_DIR=/performance

#Calculate carrier/tugboat from perf client we are running on
CLI=$(hostname | cut -d "-" -f3)
CA_NUM="${CLI: -1}"
C_NUM=${CA_NUM}
SECRET_NUM=1
# For carrier4 & carrier5 we wnt to run against their tugboats
if [[ ${C_NUM} -eq 5 ]]; then
    C_NUM=501
    SECRET_NUM=501
fi
if [[ ${C_NUM} -eq 4 ]]; then
    C_NUM=401
    SECRET_NUM=401
fi

SECRET_NAME="etcd-${SECRET_NUM}-armada-stage${CA_NUM}-south-client-tls"

export KUBECONFIG=/performance/config/carrier${C_NUM}_stgiks/admin-kubeconfig
export GOPATH=${PERF_DIR}
export METRICS_DB_KEY="${armada_performance_db_password}"

# Ensure any previous tests are uninstalled
helm uninstall etcd-rules-benchmark --namespace armada 2>/dev/null
sleep 30

# Run the tests
runBenchmark "false" "true" "false" "1" "10000"
runBenchmark "false" "true" "true" "10" "10000"
runBenchmark "false" "true" "true" "20" "10000" 

echo "Test(s) Completed"
