#!/bin/bash -e

# Position Parameters
# 1. Number of Clusters to create (defaults to 1)
# 2. Number of Workers to create on each cluster (defaults to 1)
# 3. Number of Replicas (Pods) to host applications (defaults to 1)
# 4. Cluster Prefix name (clusters wil be named $PrefixName1, $PrefixName2, etc.) (defaults to "PerfClusterAuto")
# 5. Should cluster be deleted when automation is successfully completed ("true" or "false") (defaults to "true")
# 6. Cluster worker type (defaults to u1c.2x4)
# 7. Comma separated list of Performance Test(s) to run (defaults to "" (no tests))
die() {
    printf '%s\n' "$1" >&2
    exit 1
}

containsElement() {
    local e match="$1"
    shift
    for e; do [[ "$e" == "$match" ]] && return 0; done
    return 1
}

# Ultimately will create a cluster of the requested type and poll for completion if desired, but for now just issue create request and then exit.
createCluster() {
    local clusterNamePrefix="$1"
    local clusterType="$2"
    local clusterQuantity="$3"
    local workerQuantity="$4"
    local workerFlavor="$5"
    local block="$6"
    local useAllZones="$7"
    local sendMetrics="$8"

    local wpistr="--poll-interval 30s --timeout 120m"

    if [[ -z ${clusterNamePrefix} ]]; then
        clusterNamePrefix=${cluster_prefix}
    fi

    if [[ -z ${clusterType} ]]; then
        clusterType=${cluster_type}
    fi

    if [[ -z ${clusterQuantity} ]]; then
        clusterQuantity=${num_clusters}
    fi

    if [[ -z ${workerQuantity} ]]; then
        workerQuantity=${workers}
    fi

    if [[ -z ${workerFlavor} ]]; then
        workerFlavor=${worker_type}
    fi

    # If target machine-type is a Bare Metal we want to use hardware=dedicated
    if [[ ${workerFlavor} =~ $bm_regex ]]; then
        hstr="--hardware dedicated"
    fi

    if [[ -z ${block} ]]; then
        block="true"
        sendMetrics="true"
    fi

    if [[ -z ${useAllZones} ]]; then
        useAllZones="true"
    fi

    if [[ ${block} == "false" ]]; then
        wpistr=""
    fi

    if [[ "${k8s_version}" != "default" ]]; then
        kvstr="--kube-version ${k8s_version}"
    fi

    if [[ "${large_cluster}" == "true" ]]; then
        lcstr="--pod-subnet 172.24.0.0/13"
    fi

    if [[ "${sendMetricsToDB}" == "true" ]]; then
        if [[ "${sendMetrics}" == "true" ]]; then
            metricsStr="--metrics"
        fi
    fi

    # Let's automatically try to fix up to 40% of failed worker provision/deployment attempts
    maxWorkerFailures=$((${workerQuantity} * 40 / 100))

    # VPC Cluster create API supports multiple zones. Classic equivalent doesn't (yet).
    if [[ "${clusterType}" == "classic" ]]; then
        if [[ ${#zones_array[@]} -gt 0 ]]; then
            defaultZone=${zones_array[0]}
            zoneStr="--zone ${defaultZone}"
        fi
        # Don't block on cluster create for classic multizone clusters. We need to add the additional zones separately.
        if [[ ${#zones_array[@]} -gt 1 ]]; then
            if [[ ${useAllZones} == "true" ]]; then
                wpistr=""
            fi
        fi
    else
        if [[ ${useAllZones} == "true" ]]; then
            zoneStr="--zone ${zones}"
        else
            if [[ ${#zones_array[@]} -gt 0 ]]; then
                defaultZone=${zones_array[0]}
                zoneStr="--zone ${defaultZone}"
            fi
        fi
    fi

    IFS="_" read -ra OS <<<"${operating_system^^}"
    if [[ ${clusterType} != "satellite-"* ]]; then
        case ${OS[0]} in
        "RHCOS")
            osStr="--operating-system ${OS[0]}"
            ;;
        "REDHAT" | "UBUNTU")
            osStr="--operating-system ${operating_system^^}"
            ;;
        "")
            osStr=""
            ;;
        *)
            printf "createCluster - Invalid operating system '%s'\n" "${operating_system}"
            exit 1
            ;;
        esac
        workerInfo="--workers ${workerQuantity} --max-worker-failures ${maxWorkerFailures} --machine-type ${workerFlavor} ${osStr}"
    else
        case ${OS[0]} in
        "RHCOS")
            osStr="--operating-system ${OS[0]}"
            ;;
        "REDHAT")
            osStr="--operating-system ${operating_system^^}"
            ;;
        *)
            printf "createCluster - Invalid operating system '%s'\n" "${operating_system}"
            exit 1
            ;;
        esac
        zoneStr="--zone us-south-3"
        workerInfo="--location ${location} ${osStr}"
        clusterType="satellite"
    fi

    ${perf_dir}/bin/armada-perf-client2 cluster create "${clusterType}" --name "${clusterNamePrefix}" --quantity ${clusterQuantity} --suffix ${workerInfo} ${zoneStr} ${wpistr} ${kvstr} ${metricsStr} ${lcstr} ${hstr}

    if [[ "${clusterType}" == "classic" ]]; then
        if [[ ${useAllZones} == "true" ]]; then
            for ((zi = 1; zi < ${#zones_array[@]}; zi++)); do
                zone="${zones_array[zi]}"
                ${perf_dir}/bin/armada-perf-client2 zone add ${clusterType} --zone ${zone} --worker-pool default --cluster ${clusterNamePrefix} --quantity ${clusterQuantity} --suffix
            done

            if ${block}; then
                ${perf_dir}/bin/armada-perf-client2 worker ls --cluster ${clusterNamePrefix}${clusterQuantity} --poll-interval 30s
            fi
        fi
    fi
}

cruiserMetricsController() {
    pDelay=$1
    pInterval=$2
    if [ -z "$3" ]; then
        metricsPrefix=${perftest}
    else
        metricsPrefix=$3
    fi

    if [ -n "$4" ]; then
        agg="--level=$4"
    else
        agg=""
    fi

    # - Drop "_*" suffix which is needed to handle openshift versions like 3.11_openshift and 3.11.98_1505_openshift
    IFS='.' read -r -a k8sv_array <<<"${K8S_SERVER_VERSION%%_}"
    k8s_major_version=${k8sv_array[0]}
    k8s_minor_version=${k8sv_array[1]}

    ${perf_dir}/bin/cruiser-collector --kubeconfig="${KUBECONFIG}" --delay="${pDelay}" --interval="${pInterval}" --test="${metricsPrefix}" --controlPort="${metricsControlPort}" --publish "${agg}" &

    # Wait for up to 2 minutes
    counter=1
    until [[ $counter -gt 24 ]]; do
        set +e
        resp=$(nc -zv -w10 localhost "${metricsControlPort}" 2>&1)
        set -e
        if [[ (${resp} == *"succeeded"*) ]]; then
            break
        fi
        printf "%s - %d. Waiting for Cruiser Metrics Controller to become available.\n" "$(date +%T)" "${counter}"
        sleep 5
        ((counter++))
    done
}

getInfluxdbMetricsFromPods() {
    # label of the pods which hold the influxdb results files (by default the data is expected to be in the /performance/metrics pod dir in a file of the form sysbench0.json)
    labelName=$1
    label=$2
    nameSpace=$3

    # Bring back Influxdb files from the pod with the specified label
    set +e
    printf "\n - Copy Influxdb files into ${influxdb_metrics_dir}. Process the files and send the metric data to Influxdb.\n"
    for i in $(kubectl get pods -n ${nameSpace} --no-headers -l ${labelName}=${label} | awk '{print $1}'); do
        # Remove any old Influxdb metrics files for this test case (to prevent data from multiple pods interfering with each other).
        rm -rf ${influxdb_metrics_dir}/${perftest}*.json 2>/dev/null
        printf "$i :  "
        $(kubectl cp ${nameSpace}/${i}:/performance/metrics ${influxdb_metrics_dir}) >/dev/null 2>&1
        ${perf_dir}/bin/send-file-to-Influx -metricsdir ${influxdb_metrics_dir} -testname ${perftest}
        printf "\n"
    done
    set -e
}

getInfluxdbMetricsFromJob() {
    # Label to identify the pods which hold the influxdb results files (the data is expected to be in the /performance/metrics pod dir in a file of the form sysbench0.json)
    jobName=$1
    nameSpace=$2
    # Time to wait for the metrics file to be generated before giving up. The function returns when the metrics file appears or the job itself is completes.
    waitMin=$3

    set +e
    jobPodName=$(kubectl get pod -n ${nameSpace} --no-headers -l job-name=${jobName} | awk '{print $1}')
    jobNode=$(kubectl describe pod -n ${nameSpace} ${jobPodName} | grep Node: | awk '{print $2;}')
    printf "Waiting up to ${waitMin} minutes for the metrics file to be written .. or the job to complete. \nTracking pod: ${jobPodName} in namespace: ${nameSpace} on node: ${jobNode}\n"

    counter=1
    until [[ $counter -gt ${waitMin} ]]; do
        #Check the Pod has actually started first
        status=$(kubectl get pod -n ${nameSpace} ${jobPodName} -o jsonpath='{.status.phase}')
        if [ "${status}" == "Pending" ] || [ "${status}" == "ContainerCreating" ]; then
            printf "Pod: ${jobPodName} has not started yet, will continue waiting. Current state: ${status}\n"
        else
            printf "Pod: ${jobPodName} is now in state: ${status}\n"
            # Stop waiting when the results file becomes available
            metricsFileExists=$(kubectl exec -n ${nameSpace} ${jobPodName} -- sh -c "if [ -e "/performance/metrics/${perftest}1.json" ] ; then echo \"true\"; else echo \"false\"; fi ")
            if [ "${metricsFileExists}" == "true" ]; then
                printf "Metrics file found in pod: ${jobPodName}\n"
                sleep 10 # to ensure the file is fully written
                printf "Copying metrics file to ${influxdb_metrics_dir}\n"
                kubectl cp ${perftest}/${jobPodName}:/performance/metrics ${influxdb_metrics_dir}
                ${perf_dir}/bin/send-file-to-Influx -metricsdir ${influxdb_metrics_dir} -testname ${perftest}
                break
            fi
            # Stop waiting if the job finishes
            status=$(kubectl get pod -n ${nameSpace} ${jobPodName} -o jsonpath='{.status.phase}')
            if [ "${status}" != "Running" ]; then
                printf "Stopped waiting for the metrics file as ${jobPodName} is no longing running. No metrics are available for this run. Current state: ${status} - printing logs:\n"
                kubectl logs -n ${nameSpace} ${jobPodName} --all-containers=true --ignore-errors=true
                break
            fi
        fi
        ((counter++))
        sleep 60
    done
    if [ -f "${influxdb_metrics_dir}/${perftest}1.json" ]; then
        printf "\n - Metrics file successfully copied to perf client: ${influxdb_metrics_dir}/${perftest}1.json"
    else
        printf "\n - No metrics file %s not found - so metrics will not be written to Influxdb.\n" "${influxdb_metrics_dir}/${perftest}1.json"
    fi
    set -e
}

# Performs required node labelling and tainting to configure a cluster to use edge nodes for network traffic
# See https://cloud.ibm.com/docs/containers?topic=containers-edge#edge
useEdgeNodesForNetworkTraffic() {
    local numEdgeNodes=$1
    if [[ ${numEdgeNodes} -gt 0 ]]; then
        # Remove any previous dedicated labels
        set +e
        kubectl taint nodes --overwrite=true -l dedicated=edge dedicated-
        kubectl label nodes --overwrite=true -l dedicated=edge dedicated-
        set -e

        edgeNodesPerZone=$((${numEdgeNodes} / ${#zones_array[@]}))
        labelled=0
        for ((zi = 0; zi < ${#zones_array[@]}; zi++)); do
            zone="${zones_array[zi]}"

            allZoneNodes=$(kubectl get nodes -l ibm-cloud.kubernetes.io/zone=${zone} --no-headers | awk '{print $1}')

            #Label the first 2 nodes as edge nodes
            zoneLabelled=0
            for node in $allZoneNodes; do
                kubectl label node "${node}" dedicated=edge --overwrite=true
                labelled=$((labelled + 1))
                zoneLabelled=$((zoneLabelled + 1))
                if [[ "$zoneLabelled" -eq "$edgeNodesPerZone" ]]; then break; fi
            done
            if [[ "$labelled" -eq "$numEdgeNodes" ]]; then break; fi
        done

        kubectl taint nodes -l dedicated=edge --overwrite=true dedicated=edge:NoSchedule dedicated=edge:NoExecute
        sleep 10
        # Drain the nodes we have labelled - this should ensure alb pods schedule here when deleted.
        set +e
        kubectl drain -l dedicated=edge --ignore-daemonsets --delete-emptydir-data --disable-eviction --force --timeout=600s
        kubectl uncordon -l dedicated=edge
        sleep 10

        # Force alb pods onto the edge nodes by doing an alb update.
        printf "\n - Running alb update to force alb pods onto edge nodes \n"
        ${perf_dir}/bin/armada-perf-client2 alb update --cluster "${run_on_cluster}"
        printf "Sleeping for 60s to allow the alb pods to restart\n"
        sleep 60

        # Also ensure Load Balancer pods are moved onto edge nodes by cordoning all other nodes and then deleting the pods
        kubectl cordon -l dedicated!=edge
        kubectl delete pod -n ibm-system --all
        kubectl uncordon -l dedicated!=edge
        set -e
    else
        # Remove edge node configuration
        set +e
        kubectl taint nodes --overwrite=true -l dedicated=edge dedicated-
        kubectl label nodes --overwrite=true -l dedicated=edge dedicated-
        set -e
    fi
}

httperfJMeterRun() {
    # function arguments: testType, protocol, port, startNumJMeterThreads, stopNumJMeterThreads, stepNumJMeterThreads, tput/thread(req/s) e.g. LoadBlancer http 30079 100 300 100 50
    testTypeStr=$1
    protocolStr=$2
    port=$3
    startNumThreads=$4
    stopNumThreads=$5
    stepNumThreads=$6
    threadTPutRPS=$7
    mode=$8

    testType=$(echo ${testTypeStr} | awk '{print tolower($0);}')
    protocol=$(echo ${protocolStr} | awk '{print tolower($0);}')
    export TEST_NAME="${protocol}${testType}"

    resCSVFile="${protocol}${testType}.csv"

    printf "\n%s - Starting ${mode} Jmeter ${protocol} ${testType} test against port ${port}\n" "$(date +%T)"
    # Patches the deployment to fix an issue we're seeing on edge nodes, see https://github.ibm.com/alchemy-containers/armada-performance/pull/2452
    patchDeployment
    metricsTestName=${protocol}${testType}"."nodes${totalWorkers}_replicas${totalReplicas}
    multizoneStr=
    if [[ ${#zones_array[@]} -gt 1 ]]; then
        multizoneStr=-multizone
    fi

    if [[ ${mode} == "standalone" ]]; then
        logFile=${protocol}${testType}.log
        resFile=${protocol}${testType}.jtl

        if [ -s ${hostsCSVFile} ]; then
            printf "Using the following hosts from ${hostsCSVFile}:\n"
            cat ${hostsCSVFile}
        else
            printf "\nThe ${hostsCSVFile} is empty so ${testName} will not be run\n"
            return
        fi

        printf "JMeter stepping from ${startNumThreads} threads to ${stopNumThreads} in steps of ${stepNumThreads}\n"
        numThreads=${startNumThreads}
        for ((numThreads = ${startNumThreads}; numThreads <= ${stopNumThreads}; numThreads += ${stepNumThreads})); do

            # Clean up old results files (don't error if files don't exist)
            set +e
            rm ${resFile}
            rm ${resCSVFile}
            rm ${logFile}
            set -e

            tputMins=$((threadTPutRPS * 60))

            # Metrics collection parameters : delay and interval between measurements
            cruiserMetricsController "60s" "60s" "${metricsTestName}.threads_$((numThreads))"

            # Start cruiser metrics collection
            echo -en "${METRICS_START}" | nc localhost -w 5 "${metricsControlPort}"

            jmeterCmd="${jmeterCoreParameters} -JTPUTMINS=${tputMins} -JTHREAD=${numThreads} -JPORT=${port} -JPROTOCOL=${protocol} -JHOSTS_FILE=${hostsCSVFile} -j "${logFile}" -l "${resFile}""
            printf "\nJMeter command: ${jmeterCmd}\n"
            ${jmeterCmd}

            # Stop cruiser metrics collection
            echo -en "${METRICS_STOP}" | nc localhost -w 5 "${metricsControlPort}"

            # Terminate cruiser metrics collection utility
            echo -en "${METRICS_TERMINATE}" | nc localhost -w 5 "${metricsControlPort}"

            # Convert output to a CSV
            ${jmeter_bin_dir}/JMeterPluginsCMD.sh --tool Reporter --plugin-type AggregateReport --input-jtl "${resFile}" --generate-csv "${resCSVFile}"
            cat ${resCSVFile}

            # Send to metrics service - note changing test name from ${test} to ${protocol}${testType} i.e from httppperf to httpNodePort etc.
            sendJmeterResultsReturnError -testname "${protocol}${testType}" -numThreads "${numThreads}" -resultsFile "${resCSVFile}" -metricsTestName "${metricsTestName}" ${multizoneStr} -metrics

            if [ ${stepNumThreads} -le 0 ]; then
                break
            fi

            sleep 10
        done
    else
        for ((numThreads = ${startNumThreads}; numThreads <= ${stopNumThreads}; numThreads += ${stepNumThreads})); do
            # Metrics collection parameters : delay and interval between measurements
            export KUBECONFIG=${app_kubeconfig}
            cruiserMetricsController "60s" "60s" "${metricsTestName}.threads_$((numThreads))"
            export KUBECONFIG=${load_kubeconfig}

            # Start cruiser metrics collection
            echo -en "${METRICS_START}" | nc localhost -w 5 "${metricsControlPort}"

            "${jmeter_dist_dir}/bin/run.sh" -t ${numThreads} -r ${threadTPutRPS} -d 300 >"${resCSVFile}"
            results=$(cat ${resCSVFile})

            # Stop cruiser metrics collection
            echo -en "${METRICS_STOP}" | nc localhost -w 5 "${metricsControlPort}"

            # Terminate cruiser metrics collection utility
            echo -en "${METRICS_TERMINATE}" | nc localhost -w 5 "${metricsControlPort}"

            if [[ ${results} == "FAILURE" ]]; then
                printf "\nFailed to gather all results. Metrics will not be sent\n"
            else
                cat ${resCSVFile}

                # Send metrics - note changing test name from ${test} to ${protocol}${testType} i.e from httppperf to httpNodePort etc
                sendJmeterResultsReturnError -testname "${protocol}${testType}" -numThreads "${numThreads}" -resultsFile "${resCSVFile}" -metricsTestName "${metricsTestName}" ${multizoneStr} -metrics
            fi

            sleep 10
        done
    fi
}
patchDeployment() {
    if [[ ${cluster_type} == "classic" ]]; then
        OIFS=$IFS
        IFS=$'\n'

        # Switch to target cluster
        export KUBECONFIG=${app_kubeconfig}
        kubectl get pods -A -o=wide | grep ibm-cloud-provider
        kubectl get deploy -n ibm-system -L ibm-cloud-provider-lb-app=keepalived -o=yaml

        # Get the list of deployments that we are interested in
        deployments=$(kubectl get deployments -n ibm-system | grep "ibm-cloud-provider-ip" | awk '{print $1}')

        #Repeat for each deployment
        for deployName in $deployments; do
            # First check if it already has an edge node affinity set - if it does then we won't do anything
            existingNodeAffinity=$(kubectl get deploy -n ibm-system ${deployName} -o=json | jq -rc '.spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions')
            if [[ "${existingNodeAffinity}" == *"edge"* ]]; then
                echo "Found edge node affinity already present on deployment ${deployName} - won't patch the deployment"
            else
                echo "No edge node affinity found on deployment ${deployName} - will patch the deployment"
                patchPart=$(kubectl get deploy -n ibm-system ${deployName} -o=json | jq '.spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions += [{"key": "dedicated","operator": "In","values": ["edge"]}]' | jq -rc '.spec.template.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions')
                echo "patchPart: ${patchPart}"
                fullPatch="{\"spec\":{\"template\":{\"spec\":{\"affinity\":{\"nodeAffinity\":{\"requiredDuringSchedulingIgnoredDuringExecution\":{\"nodeSelectorTerms\":[{\"matchExpressions\":${patchPart}}]}}}}}}}"
                echo "fullPatch: ${fullPatch}"
                kubectl patch deployment ${deployName} -n ibm-system -p "${fullPatch}"
            fi
        done
        IFS=$OIFS
        export KUBECONFIG="${load_kubeconfig}"
    fi
}

# Runs armada-perf-client2 commands with built in retry
apc2_with_retry() {
    apc2_command=$1
    retries=$2
    wait_time=$3
    set +e
    local counter=1

    # Support retry of temperamental commands
    until [[ ${counter} -gt ${retries} ]]; do
        if [[ ${counter} -gt 1 ]]; then
            printf "%s - %d. Command failed. Retrying.\n" "$(date +%T)" "${counter}"
        fi

        ${perf_dir}/bin/armada-perf-client2 ${apc2_command}

        if [[ $? == 0 ]]; then
            # Command was successful
            return 0
        fi

        sleep ${wait_time}
        ((counter++))
    done
    set -e
    return 1
}

waitForIngress() {
    clusterName=$1

    printf "\n%s - Checking Ingress status\n" "$(date +%T)"
    set +e
    counter=1
    ingress_ok=false
    # Wait for up to 30 min for ingressHostName to be active
    until [[ $counter -gt 6 ]]; do
        gc=$(${perf_dir}/bin/armada-perf-client2 cluster get --cluster "${clusterName}" --json)
        ih=$(echo "${gc}" | jq -r .ingress.hostname)
        isecret=$(echo "${gc}" | jq -r .ingress.secretName)

        if [[ ${ih} != "" && ${isecret} != "" ]]; then
            resp=$(nc -zv -w10 ${ih} 80 2>&1)
            if [[ (${resp} == *"succeeded"*) ]]; then
                ingress_ok=true
                break
            fi
        fi
        printf "%s - %d. Waiting for Ingress to become available.\n" "$(date +%T)" "${counter}"
        sleep 300
        ((counter++))
    done
    if [ "${ingress_ok}" != true ]; then
        printf "%s - Ingress is unavailable.\n" "$(date +%T)"
    fi
    set -e
}

# Helm 2.15 and 3.0.1 have some issues when apiservices are not available.
# See https://github.com/helm/helm/issues/6361#issuecomment-538220109
waitForAPIServicesReady() {
    maxWaitTime=$1

    curWaitTime=0
    apiservicesReady=false
    pollingInterval=60

    currentConfig=$(kubectl config current-context | cut -d '/' -f1)

    while [[ ${curWaitTime} -lt ${maxWaitTime} ]]; do
        apiservicesAvailable=$(kubectl get apiservices -A -o json)

        if [[ -z "${apiservicesAvailable}" ]]; then
            # Check for null returned from kubectl first before piping to jq when getting apiservicesNotReady
            printf "%s - Waiting for apiservices of \"%s\" config to be available.\n" "$(date +%T)" "${currentConfig}"
            sleep ${pollingInterval}
            ((curWaitTime += ${pollingInterval}))
        else
            apiservicesNotReady=$(echo ${apiservicesAvailable} | jq -j '.items[].status.conditions[] | select(.type=="Available" and .status!="True")')

            if [[ -n ${apiservicesNotReady} ]]; then
                printf "%s - Waiting for all apiservices of \"%s\" config to be available.\n" "$(date +%T)" "${currentConfig}"
                sleep ${pollingInterval}
                ((curWaitTime += ${pollingInterval}))
            else
                apiservicesReady=true
                break
            fi
        fi
    done

    if [[ "${apiservicesReady}" != true ]]; then
        printf "\n%s - Gave up waiting for all apiservices of \"%s\" config to be available.\n" "$(date +%T)" "${currentConfig}"
        printf "\n%s - CPU/Memory statistics will NOT be available.\n\n" "$(date +%T)"
    fi
}

waitForAddonReady() {
    cluster_name=$1
    maxWaitTime=$2
    addon_name=$3

    curWaitTime=0

    addonReady=false
    pollingInterval=60

    while [[ ${curWaitTime} -lt ${maxWaitTime} ]]; do
        addonCheck=$(${perf_dir}/bin/armada-perf-client2 cluster addon ls --cluster "${cluster_name}" --json | jq -r ".[] | select(.name==\"${addon_name}\") | select (.healthStatus!= null) | select (.healthStatus|contains(\"Addon Ready\"))")

        if [[ -z ${addonCheck} ]]; then
            sleep ${pollingInterval}
            ((curWaitTime += ${pollingInterval}))
        else
            addonReady=true
            break
        fi
    done

    if [[ "${addonReady}" != true ]]; then
        printf "\n%s - Gave up waiting for \"%s\" addon to be ready. Exiting.\n\n" "$(date +%T)" "${addon_name}"
        ${perf_dir}/bin/armada-perf-client2 cluster addon ls --cluster "${cluster_name}"
        return 1
    fi

    return 0
}

waitForClusterReady() {
    clusterName=$1
    maxWaitTime=$2

    curWaitTime=0

    clusterReady=false
    pollingInterval=120

    # Wait for up to 30 min for ingressHostName to be active
    while [[ ${curWaitTime} -lt ${maxWaitTime} ]]; do
        theCluster=$(${perf_dir}/bin/armada-perf-client2 cluster get --cluster "${clusterName}" --json | jq -j '.lifecycle | select(.masterStatus=="Ready")')

        if [[ -z ${theCluster} ]]; then
            sleep ${pollingInterval}
            ((curWaitTime += ${pollingInterval}))
        else
            clusterReady=true
            break
        fi
    done

    if [[ "${clusterReady}" != true ]]; then
        printf "\n%s - Gave up waiting for \"%s\" cluster to be ready. Exiting.\n\n" "$(date +%T)" "${clusterName}"
        exit 1
    fi
}

waitForNodesReady() {
    loadDriverNodes=$1
    maxWaitTime=$2

    curWaitTime=0
    nodesReady=false
    pollingInterval=60

    printf "\n%s - Checking all %s nodes of \"%s\" cluster are ready.\n" "$(date +%T)" "${loadDriverNodes}" "${load_cluster_name}"

    while [[ ${curWaitTime} -lt ${maxWaitTime} ]]; do
        nodes=$(kubectl get nodes --no-headers | grep " Ready " | wc -l)
        if ((${nodes} == ${loadDriverNodes})); then
            printf "\n%s - All nodes of \"%s\" cluster are ready.\n" "$(date +%T)" "${load_cluster_name}"
            nodesReady=true
            break
        else
            printf "%s - Waiting of for all nodes of \"%s\" cluster to be ready. Ready Nodes: %s\n" "$(date +%T)" "${load_cluster_name}" "${nodes}"
            sleep ${pollingInterval}
            ((curWaitTime += ${pollingInterval}))
        fi
    done

    if [[ "${nodesReady}" != true ]]; then
        printf "\n%s - Gave up waiting for the nodes of \"%s\" cluster to be visible via kubectl. Exiting.\n\n" "$(date +%T)" "${load_cluster_name}"
        exit 1
    fi
}

# Jmeter hits a bottleneck if we don't have many entries in the csv file - so repeat the same entries many times
generateHTTPPerfCSV() {
    test=$1
    mode=$2
    protocol=$3
    port=$4

    numCSVRepeats=200

    printf "\n%s - Generating JMeter hosts for %s %s %s testing\n\n" "$(date +%T)" "${mode}" "${protocol}" "${test}"

    export KUBECONFIG="${app_kubeconfig}"

    # Remove old hosts file if it exists
    if [[ -f ${hostsCSVFile} ]]; then
        rm ${hostsCSVFile}
    fi
    case $test in
    "nodeport")
        allNodes=$(kubectl get nodes --no-headers | awk '{print $1}')
        unset ip_array
        declare -a ip_array
        for node in $allNodes; do
            if [[ ${k8s_version} == *"openshift"* ]]; then
                ip_array+=($(kubectl get node ${node} -o jsonpath='{ $.status.addresses[?(@.type=="InternalIP")].address }'))
            else
                ip_array+=($(kubectl get node ${node} -o jsonpath='{ $.status.addresses[?(@.type=="ExternalIP")].address }'))
            fi
        done
        num_entries=${#ip_array[@]}
        for ((i = 1; i <= ${numCSVRepeats}; i++)); do
            idx=$(($i % ${num_entries}))

            if [[ ${mode} == "standalone" ]]; then
                echo ${ip_array[${idx}]} >>${hostsCSVFile}
            else
                nodeNum=$(((i - 1) % ${num_entries} + 1))
                echo "node${nodeNum},${protocol},${ip_array[${idx}]},${port}" >>${hostsCSVFile}
            fi
        done
        ;;
    "loadbalancer")
        target_ip=$(kubectl describe service httpperf-lb-service -n "${perftest}" | grep "LoadBalancer Ingress" | awk '{print $3}')
        for ((i = 1; i <= $numCSVRepeats; i++)); do
            if [[ ${mode} == "standalone" ]]; then
                echo $target_ip >>${hostsCSVFile}
            else
                echo "node1,${protocol},${target_ip},${port}" >>${hostsCSVFile}
            fi
        done
        ;;
    "loadbalancer2")
        target_ip=$(kubectl describe service httpperf-lb2-service -n "${perftest}" | grep "LoadBalancer Ingress" | awk '{print $3}')
        for ((i = 1; i <= $numCSVRepeats; i++)); do
            if [[ ${mode} == "standalone" ]]; then
                echo $target_ip >>${hostsCSVFile}
            else
                echo "node1,${protocol},${target_ip},${port}" >>${hostsCSVFile}
            fi
        done
        ;;
    "vpcApplicationLoadBalancer")
        target_hostname=$(kubectl get service -n "${perftest}" httpperf-vpc-alb-service -o jsonpath='{ $.status.loadBalancer.ingress[0].hostname }')
        for ((i = 1; i <= $numCSVRepeats; i++)); do
            if [[ ${mode} == "standalone" ]]; then
                echo $target_hostname >>${hostsCSVFile}
            else
                echo "node1,${protocol},${target_hostname},${port}" >>${hostsCSVFile}
            fi
        done
        ;;
    "vpcNetworkLoadBalancer")
        target_hostname=$(kubectl get service -n "${perftest}" httpperf-vpc-nlb-service -o jsonpath='{ $.status.loadBalancer.ingress[0].hostname }')
        for ((i = 1; i <= $numCSVRepeats; i++)); do
            if [[ ${mode} == "standalone" ]]; then
                echo $target_hostname >>${hostsCSVFile}
            else
                echo "node1,${protocol},${target_hostname},${port}" >>${hostsCSVFile}
            fi
        done
        ;;
    "ingress")
        if [[ ${useRoute} == true ]]; then
            target_ip=$(kubectl get route httpperf-ingress-service -n "${perftest}" --no-headers | awk '{print $2}')
        else
            target_ip=$(kubectl get ingress httpperf-ingress -n "${perftest}" -o json | jq -r .spec.rules[].host)
        fi
        for ((i = 1; i <= $numCSVRepeats; i++)); do
            if [[ ${mode} == "standalone" ]]; then
                echo $target_ip >>${hostsCSVFile}
            else
                echo "node1,${protocol},${target_ip},${port}" >>${hostsCSVFile}
            fi
        done
        ;;
    esac

    if [[ ${mode} == "distributed" ]]; then
        test_config_dir="${jmeter_dist_dir}/tests/httpperf"
        if [[ ! -d "${test_config_dir}" ]]; then
            mkdir "${test_config_dir}"
        fi
        mv ${hostsCSVFile} "${test_config_dir}"
    fi
}

# Delete Openshift route for test
deleteRoute() {
    # testRouteName to search for
    testRouteName=$1
    testRoutes=$(kubectl get route --all-namespaces | grep ${testRouteName} | awk '{print $1":"$2}')
    for testRoute in ${testRoutes}; do
        nsRoute=$(echo ${testRoute} | sed "s/:/ /")
        echo Deleting route: ${nsRoute}
        kubectl delete route -n ${nsRoute}
    done
}

deleteIngress() {
    # testIngressName to search for
    testIngressName=$1
    acmeairIngresses=$(kubectl get ingress --all-namespaces | grep ${testIngressName} | awk '{print $1":"$2}')
    for acmeairIngress in ${acmeairIngresses}; do
        nsIngress=$(echo ${acmeairIngress} | sed "s/:/ /")
        echo Deleting Ingress: ${nsIngress}
        kubectl delete Ingress -n ${nsIngress}
    done
}

jmeterError=0
sendJmeterResultsReturnError() {
    # The arguments for sendJmeterResultsBM:
    # ${perf_dir}/bin/sendJmeterResultsBM -testname "${perftest}" -numThreads "${ithreads}" -resultsFile "${resCSVFile}" -metricsTestName "${metricsTestName}" -metrics

    # For test with subtests or test looping through number of threads:
    #    - Add "-resetError" for the first call to reset jmeterError to 0 or set jmeterError=0 before calling this wrapper for the first time
    #    - Don't add "-resetError -failOnError" for in-between calls.  jmeterError will be updated if returnError > 0.
    #    - Add "-failOnError" for the last call to fail the test if any of the sub-tests has failed with jmeterError > 0
    #      or test can check jmetetError and fail test without calling sendJmeterResultsReturnError esp for test looping through threads.
    # For single test with just one call
    #    - Add "-failOnError" for the one call to fail the test
    if [[ $@ == *"-resetError"* ]]; then
        jmeterError=0
    fi

    sendJmeterResultsBMOptions=$(echo $@ | sed "s/-failOnError//" | sed "s/-resetError//")

    set +e
    ${perf_dir}/bin/sendJmeterResultsBM ${sendJmeterResultsBMOptions}
    returnError=$?
    set -e

    printf "\n%s Return error code is ${returnError}\n" "$(date +%T)"
    if [[ ${returnError} > 0 ]]; then
        jmeterError=${returnError}
    fi
    if [[ $@ == *"-failOnError"* ]]; then
        if [[ ${jmeterError} > 0 || ${returnError} > 0 ]]; then
            # Check jmeterError for test with subtests and returnError for single test
            printf "\n%s - Failing test with failOnError option\n" "$(date +%T)"
            exit 1
        fi
    fi
}

ensureEtcdAntiAffinity() {
    isOpenshift=$1
    clustID=$2
    carrier_name=$3
    printf "\n%s - Setting etcd anti-affinity for cluster %s on carrier %s , isOpenshift: %s \n" "$(date +%T)" "$clustID" "$carrier_name" "$isOpenshift"
    current_kubeconfig=$KUBECONFIG
    if [[ "${isOpenshift}" == "true" ]]; then
        case ${carrier_name} in
        "carrier4_stgiks")
            tugboat_name="carrier200_stgiks"
            ;;
        "carrier5_stgiks")
            tugboat_name="carrier500_stgiks"
            ;;
        esac
        # Use the Tugboat KUBECONFIG
        export KUBECONFIG=${perf_dir}/config/${tugboat_name}/admin-kubeconfig
        patchPart=$(kubectl get deploy -n master-${clustID} kube-apiserver -o=json | jq -rc '.spec.template.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution += [{"labelSelector":{"matchExpressions":[{"key":"app","operator":"In","values":["etcd"]}]},"topologyKey":"kubernetes.io/hostname"}]' | jq -rc '.spec.template.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution')
        fullPatch="{\"spec\":{\"template\":{\"spec\":{\"affinity\":{\"podAntiAffinity\":{\"requiredDuringSchedulingIgnoredDuringExecution\":${patchPart}}}}}}}"
        kubectl patch deploy -n master-${clustID} kube-apiserver -p ${fullPatch}
        kubectl rollout status deploy/kube-apiserver -n master-${clustID}
        kubectl get pods -n master-${clustID} -o=wide
    else
        # Use the Carrier/Tugboat KUBECONFIG
        if [[ ${carrier_name} == "carrier4_stgiks" ]]; then
            # Carrier4 masters now on carrier402 tugboat
            carrier_name="carrier402_stgiks"
        fi
        export KUBECONFIG=${perf_dir}/config/${carrier_name}/admin-kubeconfig
        # Add pod anti-affinity for the master pod to avoid etcd
        etcd_name="etcd-${clustID}"
        # Pod affinity is namespace specific so need to get the namespace for etcd pods
        etcd_namespace=$(kubectl get etcdcluster -A -o=json | jq ".items[] | select(.metadata.name==\"${etcd_name}\")" | jq -rc ".metadata.namespace")
        # Create and apply the patch
        patchPart=$(kubectl get deploy -n kubx-masters master-${clustID} -o=json | jq -rc ".spec.template.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution += [{\"labelSelector\":{\"matchExpressions\":[{\"key\":\"etcd_cluster\",\"operator\":\"In\",\"values\":[\"etcd-${clustID}\"]}]},\"topologyKey\":\"kubernetes.io/hostname\", \"namespaces\":[\"${etcd_namespace}\"]}]" | jq -rc '.spec.template.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution')
        fullPatch="{\"spec\":{\"template\":{\"spec\":{\"affinity\":{\"podAntiAffinity\":{\"requiredDuringSchedulingIgnoredDuringExecution\":${patchPart}}}}}}}"
        kubectl patch deploy -n kubx-masters master-${clustID} -p ${fullPatch}
        kubectl rollout status deploy/master-${clustID} -n kubx-masters
        kubectl get pods -A -o=wide | grep ${clustID}
    fi
    export KUBECONFIG=${current_kubeconfig}
}

printMasterNodes() {
    isOpenshift=$1
    clustID=$2
    carrier_name=$3

    printf "\n%s - Printing master nodes for cluster %s on carrier %s , isOpenshift: %s \n" "$(date +%T)" "$clustID" "$carrier_name" "$isOpenshift"

    current_kubeconfig=$KUBECONFIG
    if [[ "${isOpenshift}" == "true" ]]; then
        case ${carrier_name} in
        "carrier4_stgiks")
            tugboat_name="carrier200_stgiks"
            ;;
        "carrier5_stgiks")
            tugboat_name="carrier500_stgiks"
            ;;
        esac
        # Use the Tugboat KUBECONFIG
        export KUBECONFIG=${perf_dir}/config/${tugboat_name}/admin-kubeconfig
        kubectl get pods -n master-${clustID} -o=wide
    else
        # Use the Carrier/Tugboat KUBECONFIG
        if [[ ${carrier_name} == "carrier4_stgiks" ]]; then
            # Carrier4 masters now on carrier402 tugboat
            carrier_name="carrier402_stgiks"
        fi
        export KUBECONFIG=${perf_dir}/config/${carrier_name}/admin-kubeconfig
        kubectl get pods -A -o=wide | grep ${clustID}
    fi
    export KUBECONFIG=${current_kubeconfig}
}

printf "\n%s - Automation started on %s\n" "$(date +%T)" "${HOSTNAME}"

perf_dir=/performance
armada_perf_dir=${perf_dir}/armada-perf
http_scale_dir=${armada_perf_dir}/http-scale
http_perf_dir=${armada_perf_dir}/httpperf
pod_scaling_dir=${armada_perf_dir}/pod-scaling
k8s_e2e_perf_dir=${armada_perf_dir}/k8s-e2e-perf
persistent_storage_dir=${armada_perf_dir}/persistent-storage
snapshot_storage_dir=${armada_perf_dir}/persistent-storage/snapshotStorage
local_storage_dir=${armada_perf_dir}/local-storage
registry_dir=${armada_perf_dir}/registry
sysbench_dir=${armada_perf_dir}/sysbench
acmeair_dir=${armada_perf_dir}/acmeair
k8s_apiserver_dir=${armada_perf_dir}/k8s-apiserver
jmeter_dist_dir=${armada_perf_dir}/jmeter-dist
node_autoscaler_dir=${armada_perf_dir}/node-autoscaler
incluster_apiserver_dir=${armada_perf_dir}/incluster-apiserver
helm_dir=/usr/local/bin
helm_config_dir=${armada_perf_dir}/helm/config
jmeter_bin_dir=/usr/local/apache-jmeter/bin
influxdb_metrics_dir=${perf_dir}/metrics
armada_api_load_dir=${armada_perf_dir}/armada-api-load

run_auto_rc=0
metricsControlPort=10569

# Match for Bare Metal machine-types is looking for m, followed by any character, followed by a number, then c.
# e.g mb4c.20x64
bm_regex='^m.[1-9]c\.'

METRICS_START="\x1"
METRICS_STOP="\x2"
METRICS_TERMINATE="\x3"

armada_registry=stg.icr.io

export GOPATH=${perf_dir}
export METRICS_DB_KEY="${armada_performance_db_password}"
export METRICS_ROOT_OVERRIDE="${armada_performance_metrics_root_override}"

export ARGONAUTS_ARM_PERF_ALERTS_SLACK_OAUTH_TOKEN="${armada_performance_alerts_slack_token}"

case ${armada_performance_cloud_env} in
"Stage")
    export ARMADA_PERFORMANCE_API_KEY=${STAGE_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}
    export ARMADA_PERFORMANCE_INFRA_API_KEY=${PROD_GLOBAL_ARMPERF_IBMCLOUD_APIKEY} # Use production VPC Iaas for VPC based Satellite tests
    ;;
"Production")
    export ARMADA_PERFORMANCE_API_KEY=${PROD_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}
    export ARMADA_PERFORMANCE_INFRA_API_KEY=${PROD_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}
    ;;
esac

# List of tests that require a load driver cluster
declare -A DISTRIBUTED_LOAD_TESTS=(["http-perf"]=1 ["https-perf"]=1 ["olb-java"]=1 ["registry-load"]=1 ["apiserver-load"]=1)

# List of tests that require ingress to be available.
declare -A INGRESS_DEPENDENT_TESTS=(["http-perf"]=1 ["https-perf"]=1 ["olb-java"]=1 ["acmeair"]=1 ["acmeair-istio"]=1 ["acmeair-image"]=1 ["acmeair-istio-image"]=1)

# List of tests that require exclusive access to the carrier
declare -A CARRIER_LOCK_TESTS=(["incluster-apiserver"]=1 ["k8s-e2e-performance-load"]=1 ["k8s-e2e-performance-density"]=1 ["zeroworkerclusters"]=1 ["apiserver-load"]=1)

if [ "${armada_performance_sysdig_key}" = "" ]; then
    printf "\n%s - Sysdig key is NOT set - metrics will NOT be sent to sysdig\n" "$(date +%T)"
    sendToSysdig=false
else
    printf "\n%s - Sysdig key is set - metrics will be sent to sysdig\n" "$(date +%T)"
    sendToSysdig=true
fi

if [ "${armada_performance_db_password}" = "" ]; then
    printf "\n%s - db key is NOT set - metrics will NOT be sent to metrics database (Influxdb)\n" "$(date +%T)"
    sendMetricsToDB=false
else
    printf "\n%s - db key is set - metrics will be sent to metrics database (Influxdb) \n" "$(date +%T)"
    sendMetricsToDB=true
fi

# Extract the namspace suffix from the client hostname
# For example, "'dev'-mex01-perf'9'-client-01" -> "dev9"
IFS='-' read -ra host_arr <<<"${HOSTNAME}"
export ns_suffix="${host_arr[0]}${host_arr[2]: -1}"
# Assume the carrier is the matched to the perf client
export carrierName="carrier${host_arr[2]: -1}_${host_arr[0]}"
export carrierEnvName=$(sed "s/stgiks/stage/" <<< $carrierName)

num_clusters=1
num_zero_clusters=0
workers=1
host_quantity=""
replicas=0
cluster_prefix="PerfClusterAuto"
location=""
delete_cluster="false"
use_edge_nodes="false"
use_distributed_load_driver="false"
load_test_workers=12
cluster_type="classic"
worker_type="u2c.2x4"
k8s_version="default"
perf_tests=""
zones=""
kmark_cluster_size=0
apiserver_slave_pods=50
num_edge_nodes=2
large_cluster="false"
automated_run="false"

while :; do
    case $1 in
    -n | --numClusters) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            num_clusters=$2
            shift
        else
            die 'ERROR: "--numClusters" requires a non-empty option argument.'
        fi
        ;;
    -z | --numZeroClusters) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            num_zero_clusters=$2
            shift
        else
            die 'ERROR: "--numZeroClusters" requires a non-empty option argument.'
        fi
        ;;
    -w | --workers) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            workers=$2
            host_quantity=${workers}
            shift
        else
            die 'ERROR: "--workers" requires a non-empty option argument.'
        fi
        ;;
    --zones) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            zones=$2
            IFS=',' read -r -a zones_array <<<${zones}
            shift
        else
            die 'ERROR: "--zones" requires a non-empty option argument.'
        fi
        ;;
    -r | --replicas) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            replicas=$2
            shift
        else
            die 'ERROR: "--replicas" requires a non-empty option argument.'
        fi
        ;;
    -p | --clusterPrefix) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            cluster_prefix=$2
            shift
        else
            die 'ERROR: "--clusterPrefix" requires a non-empty option argument.'
        fi
        ;;
    -l | --location) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            location=$2
            shift
        else
            die 'ERROR: "--location" requires a non-empty option argument.'
        fi
        ;;
    -os | --operating-system) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            operating_system=$2

            IFS="_" read -ra OS <<<"${operating_system^^}"
            case ${OS[0]} in
            "RHCOS")
                METRICS_OS="${OS[0]}"
                ;;
            "REDHAT" | "UBUNTU")
                METRICS_OS="${operating_system^^}"
                ;;
            esac
            shift
        else
            die 'ERROR: "--operating-system" requires a non-empty option argument.'
        fi
        ;;
    -d | --deleteCluster) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            delete_cluster=$2
            shift
        else
            die 'ERROR: "--deleteCluster" requires a true or false argument.'
        fi
        ;;
    -e | --edgeNodes) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            use_edge_nodes=$2
            shift
        else
            die 'ERROR: "--edgeNodes" requires a true or false argument.'
        fi
        ;;
    -dl | --distributedLoadDriver) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            use_distributed_load_driver=$2
            shift
        else
            die 'ERROR: "--distributedLoadDriver" requires a true or false argument.'
        fi
        ;;
    -t | --workerType) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            worker_type=$2
            shift
        else
            die 'ERROR: "--workerType" requires a non-empty option argument.'
        fi
        ;;
    -ct | --clusterType) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            cluster_type=$2
            shift
        else
            die 'ERROR: "--clusterType" requires a non-empty option argument.'
        fi
        ;;
    -v | --kubeVersion) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            k8s_version=$2
            shift
        else
            die 'ERROR: "--kubeVersion" requires a non-empty option argument.'
        fi
        ;;
    -kcs | --kmarkClusterSize) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            kmark_cluster_size=$2
            shift
        else
            die 'ERROR: "--kmarkClusterSize" requires a non-empty option argument.'
        fi
        ;;
    -asr | --nodeAutoscalerTestPodReplicas) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            autoscaler_testpod_replicas=$2
            shift
        else
            die 'ERROR: "--nodeAutoscalerTestPodReplicas" requires a non-empty option argument.'
        fi
        ;;
    -asm | --nodeAutoscalerMaxNodes) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            autoscaler_maxnodes=$2
            shift
        else
            die 'ERROR: "--nodeAutoscalerMaxNodes" requires a non-empty option argument.'
        fi
        ;;
    -m | --metricsPrefix) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            export METRICS_PREFIX=$2
            shift
        else
            die 'ERROR: "--metricsPrefix" requires a non-empty option argument.'
        fi
        ;;
    -pt | --perfTests) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            perf_tests=$2
            shift
        fi
        ;;
    -asp | --apiserverSlavePods) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            apiserver_slave_pods=$2
            shift
        fi
        ;;
    -lc | --largeCluster) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            large_cluster=$2
            shift
        fi
        ;;
    -ar | --automatedRun) # Takes an option argument; ensure it has been specified.
        if [ "$2" ]; then
            automated_run=$2
            shift
        fi
        ;;
    --) # End of all options.
        shift
        break
        ;;
    -?*)
        printf 'WARN: Unknown option (ignored): %s\n' "$1" >&2
        shift
        ;;
    *) # Default case: No more options, so break out of the loop.
        break ;;
    esac

    shift
done
cluster_prefix_zero_workers=${cluster_prefix}-zero

# For Satellite clusters we need the coreos versions for host provisioning. Should had been explicity specified, but if not....
if [[ ${cluster_type} == "satellite-"* ]]; then
    if [[ ${operating_system} == "RHCOS" ]]; then
        if [[ "${k8s_version}" == "default" ]]; then
            kvstr=$(${perf_dir}/bin/armada-perf-client2 versions --json | jq -j '.openshift[] | select(.default==true) | .major, "-", .minor')
        else
            kvstr=${k8s_version%_*}
        fi
        operating_system="RHCOS_"${kvstr}
        operating_system=$(sed "s/\./_/g" <<<${operating_system})
        METRICS_OS=${operating_system^^}
    fi
fi
export METRICS_OS

printf "\n%s - Clusters: %d, Prefix: %s, ClusterType: %s, Zone(s): %s, WorkersPerZone: %d, WorkerType: %s, KubernetesVersion: %s, MetricsPrefix: %s, MetricsOS: %s\n" "$(date +%T)" "${num_clusters}" "${cluster_prefix}" "${cluster_type}" "${zones}" "${workers}" "${worker_type}" "${k8s_version}" "${METRICS_PREFIX}" "${METRICS_OS}"
if [ -n "${perf_tests}" ]; then
    printf "\n%s - Performance Tests: \"%s\"\n" "$(date +%T)" "${perf_tests}"
fi
cluster_type=$(echo ${cluster_type} | awk '{print tolower($0);}')
if [[ ${#zones_array[@]} -gt 0 ]]; then
    totalWorkers=$((${workers} * ${#zones_array[@]}))
else
    totalWorkers=${workers}
fi

# For Satellite clusters, default the location name if not specified
if [[ ${cluster_type} == "satellite-"* ]]; then
    if [[ ${location} == "" ]]; then
        location="${cluster_prefix}-loc"
    fi

    export METRICS_LOCATION="${location}"
fi

IFS=',' read -r -a tests_array <<<"${perf_tests}"
realClusters=true
if (containsElement "ZeroWorkerClusters" "${tests_array[@]}") && [ "${#tests_array[@]}" -eq "1" ]; then
    realClusters=false
fi

# If we need a distributed load cluster lets create it now in parallel with the main test cluster creation

# Set the load_cluster_name always even if use_distributed_load_driver is false
# so we can delete the driver if delete cluster is requested
load_cluster_prefix="${cluster_prefix}-load"
load_cluster_name="${load_cluster_prefix}1"

if [[ ${use_distributed_load_driver} == true ]]; then
    for test in "${tests_array[@]}"; do
        # Get test name in lower case with spaces removed
        testnospace="${test// /}"
        perftest=$(echo ${testnospace} | awk '{print tolower($0);}')

        if [ ${DISTRIBUTED_LOAD_TESTS[$perftest]+_} ]; then
            distributedLoadDriverRequired=true

            printf "\n%s - Running \"%s\" performance test. Load driver cluster required.\n" "$(date +%T)" "${perftest}"

            # Check if load driver cluster already exists, create (with no wait, if not)
            current_load_cluster=$(${perf_dir}/bin/armada-perf-client2 cluster get --cluster "${load_cluster_name}" --json | jq -j '.lifecycle | select(.masterStatus=="Ready")')
            if [[ -z ${current_load_cluster} ]]; then
                printf "\n%s - Creating Distributed Load Driver Cluster\n" "$(date +%T)"

                kvstr=""
                if [ "${k8s_version}" != "default" ]; then
                    kvstr="--kubeVersion=${k8s_version} "
                fi

                # Use a fixed number of workers in total, 1 for jmeter-master and minium of 10 for jmeter-slaves. Spread across zones
                load_test_workers_per_zone=$((${load_test_workers} / ${#zones_array[@]}))

                # Satellite clusters don't support Kubernetes load drivers. Use a classic cluster instead.
                if [[ ${cluster_type} == "satellite-"* ]]; then
                    ldct="classic"
                    ldwt="b3c.4x16"
                else
                    ldct="${cluster_type}"
                    ldwt="${worker_type}"
                fi
                # If target machine-type is a Bare Metal we want to use a VSI as load driver
                if [[ ${worker_type} =~ $bm_regex ]]; then
                    echo "Bare Metal machine-type detected will use default b3c.4x16 for load driver cluster"
                    ldwt="b3c.4x16"
                fi
                createCluster "${load_cluster_prefix}" "${ldct}" 1 ${load_test_workers_per_zone} ${ldwt} "false" "true" "false"
                load_driver_created=true
            else
                printf "\n%s - Using existing Distributed Load Driver Cluster\n" "$(date +%T)"
            fi
            break
        fi
    done
fi

if [ "$realClusters" = true ]; then

    if ((num_clusters > 0)); then
        # Check if all clusters already exist
        printf "\n%s - Checking for existing cluster(s)\n" "$(date +%T)"
        carrierClusters=$(${perf_dir}/bin/armada-perf-client2 cluster ls --json)
        maxDigits=$((${num_clusters} / 10 + 1))
        clusters_found=$(echo ${carrierClusters} | jq -r --arg cns "${cluster_prefix}\\d{1,${maxDigits}}$" '.[] | select(.name | test($cns)) | .id' | wc -l)
        normal_clusters=$(echo ${carrierClusters} | jq -r --arg cns "${cluster_prefix}\\d{1,${maxDigits}}$" '.[] | select(.name | test($cns)) | select(.state=="normal") | .id' | wc -l)

        if ((${normal_clusters} != ${num_clusters})); then
            # For Satellite clusters ensure that the location is created and control plane hosts are assigned
            IFS="-" read -ra CT <<<"${cluster_type}"

            if [[ ${CT[0]} == "satellite" ]]; then
                printf "\n%s - Preparing Satellite location '%s' using %s infrastructure\n" "$(date +%T)" "${location}" "${CT[1]}"

                # Ensure the perf-infrastructure.toml configuration file is up to date
                ${perf_dir}/bin/armada-perf-client2 sat location sync --location "${location}"

                if [[ ${perftest} == "" && ${delete_cluster} == true ]]; then
                    prepare_for_delete_flag=true
                else
                    prepare_for_delete_flag=false
                fi
                source ${armada_perf_dir}/automation/bin/prepareSatelliteLocation.sh ${location} ${automated_run} ${prepare_for_delete_flag}
                location_id=${SATELLITE_AUTOMATION_LOCATION_ID}
            else
                # Check daily tests cluster name and exit with error if cluster_prefix ends with "111"
                # to prevent unending cluster creation.
                if ((${clusters_found} > 0)); then
                    printf "\nRequired number of '%s' clusters in normal state not found. Exiting" "${cluster_prefix}"
                    exit 1
                fi
            fi

            if [ "${cluster_prefix: -3}" = "111" ]; then
                # Stop the daily test now
                printf "\nCluster prefix of ${cluster_prefix} indicates this is a cluster for daily testing on a carrier with multiple failures."
                printf "\nStopping test now and won't be creating cluster"
                printf "\nIf this is not a daily performance test cluster, please re-run the job with CLUSTER_PREFIX not ending with 111\n"
                exit 1
            fi

            if ((${clusters_found} == 0)); then
                printf "\n%s - Creating Cruiser(s)\n" "$(date +%T)"

                kvstr=""
                if [ "${k8s_version}" != "default" ]; then
                    kvstr="--kubeVersion=${k8s_version} "
                    export K8S_SERVER_VERSION="${k8s_version}"
                else
                    export K8S_SERVER_VERSION=$(${perf_dir}/bin/armada-perf-client2 versions --json | jq -j '.kubernetes[] | select(.default==true) | .major, "_", .minor')
                fi

                # Create the cluster using armada-perf-client II.
                test_start_time=$(date --iso-8601=seconds)
                createCluster
                test_end_time=$(date --iso-8601=seconds)

                # Add master/worker update bom annotations to influx for use with Grafana dshboards
                masterBOM=$(${perf_dir}/bin/armada-perf-client2 cluster get --cluster ${cluster_prefix}1 --json | jq -r '.masterKubeVersion')
                ${perf_dir}/bin/annotateBOM --carrier ${carrierEnvName} --bomVersion ${masterBOM} --bomType master --timestamp "${test_start_time}" --slack
                workerBOM=$(${perf_dir}/bin/armada-perf-client2 worker ls --cluster ${cluster_prefix}1 --json | jq -r '.[0].kubeVersion.actual')
                ${perf_dir}/bin/annotateBOM --carrier ${carrierEnvName} --bomVersion ${workerBOM} --bomType worker --timestamp "${test_end_time}" --slack

                # Get complete versions output from apc2
                ${perf_dir}/bin/armada-perf-client2 versions --json > versions.json

                # Check to see if our default platform levels have changed
                ${perf_dir}/bin/detectDefaultLevels --versionsJSON versions.json

                existing_cluster="false"
            else
                existing_cluster="true"
            fi
        else
            printf "\n%s - Using existing cluster(s)\n" "$(date +%T)"
            existing_cluster="true"
        fi

        sleep 20

        cd ${perf_dir}/config
        for ((i = 1; i <= ${num_clusters}; i = i + 1)); do
            clusterName="${cluster_prefix}${i}"
            printf "\n%s - Worker Status for Cluster %s\n" "$(date +%T)" "${clusterName}"
            ${perf_dir}/bin/armada-perf-client2 worker ls --cluster "${clusterName}"
            if [ "${existing_cluster}" != true ]; then
                rm -rf "${clusterName}"
                sleep 10
            fi
            # Ensure master pods run on different nodes to etcd pods (Currently not supported for Satellite clusters)
            if [[ ${cluster_type} != "satellite-"* ]]; then
                cluster_id=$(${perf_dir}/bin/armada-perf-client2 cluster get --cluster "${clusterName}" --json | jq -j '.id')
                isOpenshift=false
                if [[ ${k8s_version} == *"openshift"* ]]; then
                    isOpenshift=true
                fi
                ensureEtcdAntiAffinity ${isOpenshift} ${cluster_id} ${carrierName}
            fi

        done

        # For Satellite clusters we need to attach and assign some worker nodes
        # Newly created clusters will not have any workers at this point, existing clusters may exist with no workers
        if [[ ${cluster_type} == "satellite-"* ]]; then
            worker_count=$(${perf_dir}/bin/armada-perf-client2 cluster get --cluster ${clusterName} --json | jq -j '.workerCount')
            if [[ ${worker_count} -eq 0 ]]; then
                # No workers in Satellite cluster, so attach and assign them

                # Attach the Openshift cluster hosts
                source ${armada_perf_dir}/automation/bin/attachSatelliteWorkers.sh ${location_id} ${host_quantity}

                # And now assign the hosts to the Openshift cluster
                source ${armada_perf_dir}/automation/bin/assignSatelliteWorkers.sh ${location_id} ${clusterName} ${host_quantity}

                # For VPC_Gen2 Satellite clusters we need to enable public access
                if [[ ${cluster_type} == "satellite-vpc-gen2" ]]; then
                    ${armada_perf_dir}/automation/bin/enableSatPublicAccess.sh ${location}
                    printf "\n%s - DNS registration successful\n" "$(date +%T)"
                fi
            fi
        fi
    else
        # User has requested to use existing cluster, assume it exists
        existing_cluster="true"
    fi

    # Initialize array for storing Kube config files
    k8s_cfg=()
    ih_arr=()

    printf "\n%s - Getting Cluster Config File(s)\n" "$(date +%T)"

    # Initialize isROKS4Plus and clusterPlatform for default cluster
    isROKS4Plus=false
    useRoute=false
    routeParameter=""
    clusterPlatform=""
    sysdigPlatformConfig=""
    num_cluster_configs=${num_clusters}
    if [[ ${num_cluster_configs} == 0 ]]; then
        num_cluster_configs=1
    fi

    if [[ ${k8s_version} == *"openshift"* ]]; then
        # Need to get the non-admin cluster config before we can use the oc command line - we want to use the admin one - but this needs to be run.
        for ((i = 1; i <= ${num_cluster_configs}; i = i + 1)); do
            # GetClusterConfig sometimes fails even after master state is Ready, so retry for up to 10 mins
            apc2_with_retry "cluster config --cluster "${cluster_prefix}${i} 10 60
        done
        if [[ ${k8s_version} != "3."* ]]; then
            isROKS4Plus=true
            useRoute=true
            routeParameter="--route"
        fi
        clusterPlatform="Openshift"
        sysdigPlatformConfig="--openshift"

        case ${armada_performance_cloud_env} in
        "Stage")
            if [[ ${cluster_type} == "satellite-"* ]]; then
                oc_insecure_login=true
            else
                oc_insecure_login=false
            fi
            ;;
        "Production")
            oc_insecure_login=false
            ;;
        esac
    fi

    # Sleep otherwise GetClusterConfig sometimes fails
    sleep 30

    for ((i = 1; i <= ${num_cluster_configs}; i = i + 1)); do
        cluster_name="${cluster_prefix}${i}"

        cd ${perf_dir}/config
        printf "%s - Getting Cluster Config Admin File for %s\n" "$(date +%T)" "${cluster_name}"
        # GetClusterConfig sometimes fails even after master state is Ready, so retry for up to 10 mins
        apc2_with_retry "cluster config --cluster "${cluster_prefix}${i}" --admin" 10 60

        # Get the kubeconfig info
        cluster_config_dir="${perf_dir}"/config/"${cluster_name}"
        if [[ ! -d "${cluster_config_dir}" ]]; then
            mkdir -p "${cluster_config_dir}"
            mv "kubeConfigAdmin-${cluster_name}.zip" "${cluster_config_dir}"
            unzip -o -j -d "${cluster_config_dir}" "${cluster_config_dir}"/kubeConfigAdmin-"${cluster_name}".zip

            # Kubernetes E2E tests need the path of the certifcate files to be explicity specified
            conf_yaml=$(ls ${cluster_config_dir}/*.yml)
            export KUBECONFIG=${conf_yaml}

            sed -i 's|\(certificate-authority: \)|\1'${cluster_config_dir}'/|' ${conf_yaml}
            sed -i 's|\(client-certificate: \)|\1'${cluster_config_dir}'/|' ${conf_yaml}
            sed -i 's|\(client-key: \)|\1'${cluster_config_dir}'/|' ${conf_yaml}
        fi

        conf_yaml=$(ls ${cluster_config_dir}/*.yml)
        export KUBECONFIG=${conf_yaml}

        # For Satellite clusters in stage, we need to ensure we avoid issues with self-signed certificate
        if [[ ${cluster_type} == "satellite-"* ]]; then
            sed -i '/server:/a \ \ \ \ insecure-skip-tls-verify: true' ${conf_yaml}
        fi

        # Wait long enough for openvpn to complete if necessary, as this causes the master to be unavailable.
        waitForAPIServicesReady 600

        # Add KUBECONFIG file to array
        k8s_cfg+=(${KUBECONFIG})

        # Enable logDNA if logDNA ingestion key is set
        if [ "${armada_performance_logdna_ingestion_key}" != "" ]; then
            printf "\n%s - LogDNA ingestion key is set - Enabling logDNA\n" "$(date +%T)"
            ${armada_perf_dir}/automation/bin/enableLogDNA.sh ${clusterPlatform}
        fi

        if [[ ${isROKS4Plus} == true ]]; then
            # Reduce Prometheus resource requests otherwise it affects pod placement
            oc apply -f ${armada_perf_dir}/openshift/config/os-monitoring-config.yaml
        fi
    done

    if [[ ${distributedLoadDriverRequired} == true ]]; then
        printf "\n%s - Waiting for distributed load driver\n" "$(date +%T)"
        waitForClusterReady "${load_cluster_name}" 1800

        load_cluster_id=$(${perf_dir}/bin/armada-perf-client2 cluster get --cluster "${load_cluster_name}" --json | jq -j '.id')

        printf "\n%s - Getting Load Driver Cluster Config File(s)\n" "$(date +%T)"
        cd ${perf_dir}/config

        # GetClusterConfig sometimes fails even after master state is Ready, so retry for up to 10 mins
        apc2_with_retry "cluster config --cluster "${load_cluster_name}" --admin" 10 60

        # Get the kubeconfig info
        load_cluster_config_dir="${perf_dir}"/config/"${load_cluster_name}"
        if [[ ! -d "${load_cluster_config_dir}" ]]; then
            mkdir -p "${load_cluster_config_dir}"
        fi

        mv "kubeConfigAdmin-${load_cluster_name}.zip" "${load_cluster_config_dir}"
        unzip -o -j -d "${load_cluster_config_dir}" "${load_cluster_config_dir}/kubeConfigAdmin-${load_cluster_name}.zip"

        load_conf_yaml=$(ls ${load_cluster_config_dir}/*.yml)

        export KUBECONFIG="${load_conf_yaml}"
        waitForNodesReady ${load_test_workers} 5400 # Wait for up to 90 minutes

        # Kube 1.15+ issues - wait for up to 30 minutes until all apiservices are ready.
        printf "\n%s - Checking apiservices are ready\n" "$(date +%T)"
        waitForAPIServicesReady 1800

        completedName=$(echo "cluster-config-complete-${load_cluster_id}" | tr '[:upper:]' '[:lower:]') # Just in case cluster Ids ever contain upper case characters
        setupCompleted=$(kubectl get configmap ${completedName} --ignore-not-found | wc -l)

        if [[ ${load_driver_created} || ${setupCompleted} -eq 0 ]]; then
            if [[ ${k8s_version} == *"openshift"* ]]; then
                # Need to get the non-admin cluster config before we can use the oc command line - we want to use the admin one - but this needs to be run.
                ${perf_dir}/bin/armada-perf-client2 cluster config --cluster "${load_cluster_name}"
                # The previous call triggers some async actions, need to wait or next step can fail
                sleep 180
                source ${armada_perf_dir}/automation/bin/enableRootUserOnOpenshift.sh default ${oc_insecure_login}
            fi
            # Ensure secrets exist
            source ${armada_perf_dir}/automation/bin/setupRegistryAccess.sh default

            if [[ ${isROKS4Plus} == true ]]; then
                # Reduce Prometheus resource requests otherwise it affects pod placement
                oc apply -f ${armada_perf_dir}/openshift/config/os-monitoring-config.yaml
            fi

            # Mark cluster config as being complete
            kubectl create configmap ${completedName} --from-literal=status=complete
        fi

        # Enable logDNA for the load driver if logDNA ingestion key is set
        if [ "${armada_performance_logdna_ingestion_key}" != "" ]; then
            printf "\n%s - LogDNA ingestion key is set - Enabling logDNA\n" "$(date +%T)"
            ${armada_perf_dir}/automation/bin/enableLogDNA.sh ${clusterPlatform}
        fi
        # To allow some time for the load driver cluster to be ready
        sleep 2m
        printf "\n%s - Distributed load driver is ready\n" "$(date +%T)"

        set +e
        printf "\n%s - Printing worker node processors for Load Driver cluster\n" "$(date +%T)"
        source ${armada_perf_dir}/automation/bin/getWorkerProcs.sh
        printf "\n%s - Distributed load driver is ready\n" "$(date +%T)"
        set -e
    fi

    # If we're not creating any clusters, then assume we've already created one.
    # For now, always run tests against a single (the first) cluster
    run_on_cluster="${cluster_prefix}1"

    cluster_config_dir="${perf_dir}"/config/"${run_on_cluster}"
    conf_yaml=$(ls ${cluster_config_dir}/*.yml)
    export KUBECONFIG=${conf_yaml}

    # For Satellite clusters in stage, we need to ensure we avoid issues with self-signed certificate
    # Note we already do a login when the cluster is created - but that can expire, so login each time a test is run.
    if [[ ${cluster_type} == "satellite-"* ]]; then
        set +e
        insecure_set=$(grep insecure-skip-tls-verify ${conf_yaml})
        if [[ -z ${insecure_set} ]]; then
            sed -i '/server:/a \ \ \ \ insecure-skip-tls-verify: true' ${conf_yaml}
        fi
        set -e
    fi

else
    printf "\n%s - Only zero worker test was requested so no real clusters will be created\n" "$(date +%T)"
fi

if [[ -n "${perf_tests}" ]]; then
    if [ "$realClusters" = true ]; then

        # If not overridden by user
        if (("${replicas}" == 0)); then
            printf "\n%s - Determine replicas\n" "$(date +%T)"

            # Set the number of replicas to the number of nodes
            # We used to set it to (number of available cores - 1) * number of workers
            cpu_capacity=$(kubectl get nodes -o jsonpath='{.items[*].status.capacity.cpu}')

            IFS=' ' read -r -a nodecpu_array <<<${cpu_capacity}
            replicas=${#nodecpu_array[@]}
        fi

        # Check if any of the tests require Ingress and wait for its availability if so
        ingressRequired=false
        for test in "${tests_array[@]}"; do
            # Get test name in lower case with spaces removed
            testnospace="${test// /}"
            perftest=$(echo ${testnospace} | awk '{print tolower($0);}')

            if [ ${INGRESS_DEPENDENT_TESTS[$perftest]+_} ]; then
                ingressRequired=true
                break
            fi
        done

        # For tests that use ingress, wait for the ingress hostname to be available
        if [[ ${ingressRequired} == true ]]; then
            waitForIngress ${run_on_cluster}
            # Perform ingress tuning on IKS clusters
            if [[ ${isROKS4Plus} != true ]]; then
                # Tune Ingress to set the maximum number of requests that can be served through one keep-alive connection.
                printf "\n%s - Performing Ingress tuning on ${run_on_cluster}\n" "$(date +%T)"
                kubectl patch configmap/ibm-k8s-controller-config -n kube-system --type merge -p '{"data":{"keep-alive-requests":"100000"}}'
            fi
        fi

        # Get the cluster id and master kube version
        gc=$(${perf_dir}/bin/armada-perf-client2 cluster get --cluster "${run_on_cluster}" --json)
        clusterID=$(echo "${gc}" | jq -r .id)
        mkv=$(echo "${gc}" | jq -r .masterKubeVersion)

        # Kube 1.15 issues - wait for up to 30 minutes until all apiservices are ready.
        printf "\n%s - Checking apiservices are ready\n" "$(date +%T)"
        waitForAPIServicesReady 1800

        set +e
        printf "\n%s - Printing worker node processors for Target cluster\n" "$(date +%T)"
        source ${armada_perf_dir}/automation/bin/getWorkerProcs.sh
        printf "\n%s - Printing Cluster and worker node info for Target cluster\n" "$(date +%T)"
        ${perf_dir}/bin/armada-perf-client2 cluster get --cluster "${run_on_cluster}"
        ${perf_dir}/bin/armada-perf-client2 worker ls --cluster "${run_on_cluster}"
        kubectl get nodes -o=wide

        if [[ ${cluster_type} != "satellite-"* ]]; then
            isOpenshift=false
            if [[ ${k8s_version} == *"openshift"* ]]; then
                isOpenshift=true
            fi
            printf "\n%s - Printing master node information for Target cluster\n" "$(date +%T)"
            printMasterNodes ${isOpenshift} ${clusterID} ${carrierName}
        fi
        set -e

        # Install or remove the sysdig agent pod
        sysdig_agent_pod="sysdig-agent"
        sysdig_agent_ns="ibm-observe"
        sysdig_agent_pods_running=($(kubectl get pods -n ${sysdig_agent_ns} | grep ${sysdig_agent_pod} | grep "Running" | wc -l))
        if ${sendToSysdig}; then
            if (("${sysdig_agent_pods_running}" > 0)); then
                printf "\n%s - ${sysdig_agent_pod} (sysdig supplied monitoring agent) is already running and will not be restarted\n" "$(date +%T)"
            else
                printf "\n%s - ${sysdig_agent_pod} (sysdig supplied monitoring agent) is not running and will be created\n" "$(date +%T)"
                set +e
                # Need to login to IBM Cloud for openshift to work here
                if [[ "${isOpenshift}" == "true" ]]; then
                    PERF_METADATA_TOML=${perf_dir}/armada-perf/armada-perf-client2/config/perf-metadata.toml
                    IKS_ENDPOINT=$(/performance/bin/tomlToJson $PERF_METADATA_TOML | jq -r '.iks.endpoint')
                    API_ENDPOINT=$(/performance/bin/tomlToJson $PERF_METADATA_TOML | jq -r '.ibmcloud.iam_endpoint' | cut -d '.' -f2-)
                    REGION="us-south"
                    export IBMCLOUD_API_KEY=${ARMADA_PERFORMANCE_API_KEY}
                    ibmcloud login -a ${API_ENDPOINT} -r ${REGION}
                fi
                printf "\n%s - Delete any previous ${sysdig_agent_pod} daemon-sets: " "$(date +%T)"
                kubectl delete daemonset -n ${sysdig_agent_ns} ${sysdig_agent_pod} 2>/dev/null
                printf "\n%s - Installing ${sysdig_agent_pod}\n" "$(date +%T)"
                sudo rm /tmp/sysdig-agent-*
                curl -sL https://ibm.biz/install-sysdig-k8s-agent | bash -s -- -a "${armada_performance_sysdig_key}" -c "ingest.us-south.monitoring.cloud.ibm.com" -ac "sysdig_capture_enabled: false" --tags "auto_perfClient:${HOSTNAME},auto_carrier:${carrierName},auto_K8sVersion:${k8s_major_version}.${k8s_minor_version}" ${sysdigPlatformConfig}
                set -e
                printf "\n%s - Sleeping for 20s to allow the sysdig_agent pods to start\n" "$(date +%T)"
                sleep 20
            fi
        else
            if (("${sysdig_agent_pods_running}" > 0)); then
                printf "\n%s - sysdig was not requested but pods are running. They will be stopped.\n" "$(date +%T)"
                set +e
                kubectl delete daemonset -n ${sysdig_agent_ns} ${sysdig_agent_pod} 2>/dev/null
                set -e
            fi
        fi
    else
        printf "\n%s - Only zero worker test was requested so no Ingress etc required\n" "$(date +%T)"
    fi

    # Save the cruiser kubernetes version in an env var for use by the metrics gathering library
    export K8S_SERVER_VERSION="${mkv}"
    IFS='.' read -r -a k8sv_array <<<"${K8S_SERVER_VERSION%%_}"
    export K8S_MAJOR_VERSION=${k8sv_array[0]}
    export K8S_MINOR_VERSION=${k8sv_array[1]}

    osstr=$(${perf_dir}/bin/armada-perf-client2 worker-pool get --cluster "${run_on_cluster}" --worker-pool default --json | jq -r '.operatingSystem')
    if [[ -n ${osstr} ]]; then
        export METRICS_OS="${osstr^^}"
        printf "\n%s - Setting the metrics Operating System from 'default' workerpool : %s\n" "$(date +%T)" "${METRICS_OS}"
    fi

    # Combine K8S E2E load and density tests into a single run
    # Note that this will result in the K8S E2E tests running last
    if containsElement "K8s-E2e-Performance-Density" "${tests_array[@]}" && containsElement "K8s-E2e-Performance-Load" "${tests_array[@]}"; then
        tests_array=("${tests_array[@]/'K8s-E2e-Performance-Density'/}")
        tests_array=("${tests_array[@]/'K8s-E2e-Performance-Load'/}")
        tests_array+=('K8s-E2e-Performance')
    fi

    if containsElement "K8s-E2e-Scalability-Density" "${tests_array[@]}" && containsElement "K8s-E2e-Scalability-Load" "${tests_array[@]}"; then
        tests_array=("${tests_array[@]/'K8s-E2e-Scalability-Density'/}")
        tests_array=("${tests_array[@]/'K8s-E2e-Scalability-Load'/}")
        tests_array+=('K8s-E2e-Scalability')
    fi
    k8s_e2e_perf_extra_args=""

    # Remove blank elements from tests array
    tmp_array=()
    for i in "${tests_array[@]}"; do
        if [ -n "${i}" ]; then
            tmp_array+=("${i}")
        fi
    done
    tests_array=("${tmp_array[@]}")
    unset tmp_array

    for test in "${tests_array[@]}"; do
        printf "\n%s - Running tests\n" "$(date +%T)"
        # Get test name in lower case with spaces removed
        testnospace="${test// /}"
        perftest=$(echo ${testnospace} | awk '{print tolower($0);}')

        if [ "$realClusters" = true ]; then

            printf "\n%s - Setting up registry access for %s\n" "$(date +%T)" "${test}"
            source ${armada_perf_dir}/automation/bin/setupRegistryAccess.sh "${perftest}"
            source ${armada_perf_dir}/automation/bin/enableRootUserOnOpenshift.sh "${perftest}" ${oc_insecure_login}

            set +e
            # Delete any previous deployment
            printf "\n%s - Deleting any previous deployment\n" "$(date +%T)"
            ${helm_dir}/helm uninstall "${perftest}" --namespace "${perftest}" 2>/dev/null
            set -e
            # Give it a chance to sleep
            sleep 60

        fi

        printf "\n%s - Running \"%s\" test\n" "$(date +%T)" "${test}"

        export TEST_NAME="${perftest}"

        # Geneate a port for the cuiser metrics controller based on cluster prefix and perftest - to avoid clash if running multiple tests
        metricsControlPort=$(echo 3$((0x$(echo "${cluster_prefix}${perftest}" | md5sum | cut -f 1 -d " " | cut -c 1-5))) | cut -c 1-5)

        # Remove any old Influxdb metrics files
        rm -rf ${influxdb_metrics_dir}/${perftest}*.json 2>/dev/null

        # Remove any edge node taints just in case they are left behind by previous test
        useEdgeNodesForNetworkTraffic 0

        if [[ ${cluster_type} != "satellite-"* ]] && [ ${CARRIER_LOCK_TESTS[$perftest]+_} ]; then
            # Take a lock so other perf clients won't hit the masters hard
            # Note Satellite does not need to lock as the masters are not running on the carrier
            ${perf_dir}/bin/carrier-lock --kubeconfig=${perf_dir}/config/${carrierName}/admin-kubeconfig --action acquire --max-wait-time=120m
        fi
        case "${perftest}" in
        "add-workers")
            workerPoolName="default"
            wpgl=$(${perf_dir}/bin/armada-perf-client2 worker-pool get --cluster "${run_on_cluster}" --worker-pool "${workerPoolName}" --json)
            currentSize=$(echo ${wpgl} | jq -r .workerCount)
            newSize=$((currentSize + workers))
            printf "\n%s - Increasing worker pool size from %d to %d\n" "$(date +%T)" "${currentSize}" "${newSize}"
            ${perf_dir}/bin/armada-perf-client2 worker-pool resize --cluster "${run_on_cluster}" --worker-pool "${workerPoolName}" --size-per-zone ${newSize} --poll-interval 15s --metrics

            printf "\n%s - Decreasing worker pool size from %d to %d\n" "$(date +%T)" "${newSize}" "${currentSize}"
            ${perf_dir}/bin/armada-perf-client2 worker-pool resize --cluster "${run_on_cluster}" --worker-pool "${workerPoolName}" --size-per-zone ${currentSize} --poll-interval 15s --metrics
            ;;

        "kubemark")
            export KUBE_ROOT="${GOPATH}"/src/k8s.io/kubernetes

            export CLOUD_PROVIDER=iks

            export KUBEMARK_IMAGE_REGISTRY="${armada_registry}"
            export KUBEMARK_IMAGE_TAG="${k8s_version}"

            # See https://github.com/kubernetes/kubernetes/issues/69735
            export HOLLOW_PROXY_TEST_ARGS="--use-real-proxier=false"

            export USE_EXISTING="y"
            export ISBUILD="N"
            export ISCLEAN="N"
            export ISFED="N/A"
            export DESIRED_NODES="${kmark_cluster_size}"
            export KUBEMARK_NUM_NODES="${kmark_cluster_size}"

            export custom_spawn_cluster_prefix="hollow_node_pods_cluster_auto"
            export custom_spawn_cluster="${custom_spawn_cluster_prefix}1"

            spawn_cluster=$(${perf_dir}/bin/armada-perf-client2 cluster get --cluster "${custom_spawn_cluster}" --json | jq -r .state)
            if [[ ${spawn_cluster} != "normal" ]]; then
                printf "\n%s - Creating cluster to host hollow node pods\n" "$(date +%T)"

                kvstr=""
                if [ "${k8s_version}" != "default" ]; then
                    kvstr="--kubeVersion=${k8s_version} "
                fi

                # 1 x b3c.4x16 workers supports ~40 hollow nodes for cluster sizes <= 1000 and ~33 for cluster sizes > 1000
                # (hollow proxy has higher cpu requests once the number of nodes exceeds 1,000)
                if ((${kmark_cluster_size} > 1000)); then
                    actual_spawn_nodes="$((((${kmark_cluster_size} - 1) / 33) + 1))"
                else
                    actual_spawn_nodes="$((((${kmark_cluster_size} - 1) / 40) + 1))"
                fi

                set -x
                if [[ ${cluster_type} == "classic" ]]; then
                    hnwf="b3c.4x16"
                else
                    hnwf="b2.4x16"
                fi

                createCluster "${custom_spawn_cluster_prefix}" "${cluster_type}" 1 "${actual_spawn_nodes}" "${hnwf}" "true" "false" "false"
                set +x
            fi

            printf "\n%s - Getting Hollow Node hosting cluster Config File(s)\n" "$(date +%T)"
            cd ${perf_dir}/config
            ${perf_dir}/bin/armada-perf-client2 cluster config --cluster "${custom_spawn_cluster}" --admin
            sleep 30

            # Get the cluster config for our hollow node pod hosting cluster
            spawn_cluster_config_dir="${perf_dir}"/config/"${custom_spawn_cluster}"
            if [[ -d "${spawn_cluster_config_dir}" ]]; then
                rm -r "${spawn_cluster_config_dir}"
            fi
            mkdir -p "${spawn_cluster_config_dir}"
            mv "${perf_dir}"/config/kubeConfigAdmin-"${custom_spawn_cluster}".zip "${perf_dir}"/config/"${custom_spawn_cluster}"
            unzip -o -j -d "${spawn_cluster_config_dir}" "${spawn_cluster_config_dir}"/kubeConfigAdmin-"${custom_spawn_cluster}".zip

            # Kubernetes E2E tests need the path of the certifcate files to be explicity specified
            spawn_conf_yaml=$(ls ${spawn_cluster_config_dir}/*.yml)

            sed -i 's|\(certificate-authority: \)|\1'${spawn_cluster_config_dir}'/|' ${spawn_conf_yaml}
            sed -i 's|\(client-certificate: \)|\1'${spawn_cluster_config_dir}'/|' ${spawn_conf_yaml}
            sed -i 's|\(client-key: \)|\1'${spawn_cluster_config_dir}'/|' ${spawn_conf_yaml}

            spawn_conf_yaml=$(ls ${spawn_cluster_config_dir}/*.yml)
            export SPAWN_KUBECONFIG=${spawn_conf_yaml}

            export CUSTOM_MASTER_CONFIG="${KUBECONFIG}"
            export CUSTOM_SPAWN_CONFIG="${SPAWN_KUBECONFIG}"

            # Need to avoid kube-dns pods getting scheduled onto the hollow node - let's disable the autoscaler.
            kubectl -n kube-system get deployment | grep coredns-autoscaler
            if [[ $? -eq 0 ]]; then
                kubectl scale deployment -n kube-system coredns-autoscaler --replicas=0
            fi

            # Fire up the hollow nodes
            cd "${KUBE_ROOT}"
            sudo -E ./test/kubemark/start-kubemark.sh

            # Run the standard kubernetes performance tests against Kubemark cluster
            # (But first we need to cordon the real nodes. This ensures k8s e2e test configuraiton is based on hollow nodes)
            kubectl get nodes --no-headers | awk -F" " '{print $1}' | grep "10." | xargs -L 1 kubectl cordon
            perftest=k8s-e2e-performance
            k8s_e2e_perf_extra_args="--allowed-not-ready-nodes=10000 --system-pods-startup-timeout=10s"
            metricsControlPort=0
            ;&
        "k8s-e2e-performance-load" | "k8s-e2e-performance-density")
            export KUBE_ROOT="${GOPATH}"/src/k8s.io/perf-tests/clusterloader2
            export KUBE_MASTER_IP=$(sed -n '/server:/p' "${KUBECONFIG}" | rev | cut -d/ -f1 | rev | head -1)
            export KUBE_MASTER=$(echo ${KUBE_MASTER_IP} | cut -d: -f1)

            export LOG_DUMP_SSH_KEY="${cluster_config_dir}"/.ssh/id_rsa
            export LOG_DUMP_SSH_USER=root
            send_metrics="-metrics"

            cd "${KUBE_ROOT}"
            printf "%s - Building Clusterloader....\n" "$(date +%T)"
            sudo -E go build -o clusterloader './cmd/'
            printf "%s - Clusterloader build completed.\n" "$(date +%T)"

            # Create the directories needed for the logs and reports
            k8s_report_dir="${k8s_e2e_perf_dir}"/reports
            rm -rf "${k8s_report_dir}"
            mkdir -p "${k8s_report_dir}"

            k8s_log_dir="${k8s_e2e_perf_dir}"/logs
            rm -rf "${k8s_log_dir}"
            mkdir -p "${k8s_log_dir}"

            case "${perftest}" in
            "k8s-e2e-performance-load")
                test_config_dir="${KUBE_ROOT}/testing/load"
                send_metrics="-metrics"
                ;;
            "k8s-e2e-performance-density")
                test_config_dir="${KUBE_ROOT}/testing/density"
                send_metrics="-metrics"
                ;;
            esac

            if [[ "${metricsControlPort}" != 0 ]]; then
                # As the number of nodes in a cluster increaes, so does the length of time it takes the test to run.
                # So, adjust the metrics collection interval to be 60s for every 200 nodes.
                numNodes=$(kubectl get nodes --no-headers | wc -l)
                metricInterval="$((((${numNodes} / 200) * 60) + 60))"

                cruiserMetricsController "10s" "${metricInterval}s" "${perftest}" "namespace"

                # Start gathering cruiser metrics
                echo -en "${METRICS_START}" | nc localhost -w 5 "${metricsControlPort}"
            fi

            # Kubernetes E2E testing will return with a non-zero exit code if the tests do not pass. We want to save this exit code but carry on processing
            set +e
            # Use the Kubernetes master API server local proxy
            master_internal_ip=172.21.0.1

            # We need to create the namespace before the test so that we can create the secret in it

            kubectl create namespace monitoring
            test_start_time=$(date --iso-8601=seconds)

            # Run the test using clusterloader on a large (100 or larger) cluster and get the results from prometheus
            # This uses overrides_large.yaml. If a small 5 node cluster is needed use overides.yaml.

            if [[ ${workers} -eq 5 ]]; then
                overrides="${test_config_dir}/overrides.yaml"
            else
                overrides="${test_config_dir}/overrides_large.yaml"
            fi

            "${KUBE_ROOT}"/clusterloader --kubeconfig "${KUBECONFIG}" --v=2 --alsologtostderr --provider skeleton --report-dir "${k8s_report_dir}" --log_dir "${k8s_log_dir}" --testconfig "${test_config_dir}/config.yaml" --testoverrides "${overrides}" --enable-prometheus-server --tear-down-prometheus-server=false --prometheus-scrape-etcd=false --prometheus-scrape-kube-proxy=false --prometheus-scrape-kubelets=false --prometheus-scrape-node-exporter=false --master-internal-ip ${master_internal_ip}
            if [[ $? -gt 0 ]]; then
                run_auto_rc=$?
            fi
            test_end_time=$(date --iso-8601=seconds)
            set -e

            # Start the prometheus port-forward to enable getting results via curl
            nohup kubectl --namespace monitoring port-forward svc/prometheus-k8s 9090 --address=0.0.0.0 &
            sleep 5

            # Curl command to get the overall response time
            date=$(date "+%F-%T")

            # Curl command to get the overall response times
            curl -g 'http://127.0.0.1:9090/api/v1/query?query=sum(rate(apiserver_request_duration_seconds_sum{resource!="events",verb!~"WATCH|WATCHLIST|PROXY|proxy|CONNECT"}[5m]))%20by%20(resource,subresource,scope,verb)%20/%20sum(rate(apiserver_request_duration_seconds_count{resource!="events",verb!~"WATCH|WATCHLIST|PROXY|proxy|CONNECT"}[5m]))%20by%20(resource,subresource,scope,verb)' | jq '.' >${k8s_report_dir}/APIResponsivenessOverall_load_${date}.json

            # End the port-forward
            sudo pkill -u jenkins -f "port-forward svc/prometheus-k8s"

            if [[ "${metricsControlPort}" != 0 ]]; then
                # Stop gathering cruiser metrics
                echo -en "${METRICS_STOP}" | nc localhost -w 5 "${metricsControlPort}"
            fi

            sudo chown -R jenkins:"Domain Users" "${k8s_report_dir}"

            # Process k8s metrics and send to IBM Cloud metics service
            sleep 5
            ${perf_dir}/bin/kubernetes-e2e -verbose -resultsDir="${k8s_report_dir}" -kubeconfig="${KUBECONFIG}" "${send_metrics}"

            if [[ "${metricsControlPort}" != 0 ]]; then
                # Terminate cruiser metrics collector utility
                echo -en "${METRICS_TERMINATE}" | nc localhost -w 5 "${metricsControlPort}"
            fi
            ;;
        "k8s-netperf")
            # Measures pod -> pod communication performance with and without host networking

            # N.B. Enabling host networking on the pod allows applications running in the pod to directly see the network interfaces
            # of the host machine where the pod was started.
            mkdir -p /tmp/kubernetes
            cd "$_"

            # Ensure we're starting from a clean environment
            kubectl delete rc --all -n "${perftest}" --ignore-not-found=true
            kubectl delete services --all -n "${perftest}" --ignore-not-found=true

            netperf_image_name="${armada_registry}/armada_performance_${ns_suffix}/${perftest}"

            cruiserMetricsController "30s" "120s" "${perftest}" "pod"

            # Run 1 : First run with host networking disabled
            echo -en "${METRICS_START}" | nc localhost -w 5 "${metricsControlPort}"
            ${perf_dir}/bin/netperf -kubeConfig="${KUBECONFIG}" -image="${netperf_image_name}"
            echo -en "${METRICS_STOP}" | nc localhost -w 5 "${metricsControlPort}"

            ${perf_dir}/bin/kubernetes-netperf -verbose -resultsFile="${perftest}-latest.csv" -metrics # Send metrics

            # Run 2 : Now run with host networking enabled
            echo -en "${METRICS_START}" | nc localhost -w 5 "${metricsControlPort}"
            ${perf_dir}/bin/netperf -kubeConfig="${KUBECONFIG}" -image="${netperf_image_name}" -hostnetworking
            echo -en "${METRICS_STOP}" | nc localhost -w 5 "${metricsControlPort}"

            ${perf_dir}/bin/kubernetes-netperf -verbose -resultsFile="${perftest}-latest.csv" -metrics # Send metrics

            # Terminate cruiser metrics collection utility
            echo -en "${METRICS_TERMINATE}" | nc localhost -w 5 "${metricsControlPort}"
            ;;
        "http-scale")
            # Length of http-scale test in seconds
            httpScaleRunLengthSec=600

            # Containers need a duration in seconds, so add s before passing it to the containers.
            httpScaleRunDuration=${httpScaleRunLengthSec}"s"

            # Ensure we're starting from a clean environment
            kubectl delete rc --all -n "${perftest}" --ignore-not-found=true
            kubectl delete services --all -n "${perftest}" --ignore-not-found=true

            printf "\n%s - Installing http-scale on ${run_on_cluster}\n" "$(date +%T)"
            cd ${http_scale_dir}

            #Add parameters for metrics
            kubectl create configmap metrics-name-data --from-literal=prefix=${METRICS_PREFIX} --from-literal=k8s-version=${K8S_SERVER_VERSION} --from-literal=test-name=${test} --from-literal=run-length=${httpScaleRunDuration} -n "${perftest}"

            # Update image names with namespace
            sed -e "s:/armada_performance/:/armada_performance_${ns_suffix}/:" <template-aggregator-rc.yaml >aggregator-rc.yaml
            sed -e "s:/armada_performance/:/armada_performance_${ns_suffix}/:" <template-vegeta-rc.yaml >vegeta-rc.yaml
            sed -e "s:/armada_performance/:/armada_performance_${ns_suffix}/:" <template-nginx-rc.yaml >nginx-rc.yaml

            # Run bootstrap script to set up test
            ./bootstrap.sh

            # Metrics collection parameters : delay and interval between measurements
            cruiserMetricsController "60s" "60s" "${perftest}" "pod"

            # Start cruiser metrics collection
            echo -en "${METRICS_START}" | nc localhost -w 5 "${metricsControlPort}"

            #scale number of webservers. Each 1 core webserver should be able to handle 10,000 requests/s
            kubectl scale rc nginx --replicas=${workers} -n "${perftest}"

            # Scale number of vegeta clients. Each replica delivers 1,000 requests/s
            vegeta_pods_per_node=4
            final_vegeta_pods=$((${workers} * ${vegeta_pods_per_node}))
            kubectl scale rc vegeta --replicas=${final_vegeta_pods} -n "${perftest}"

            # ...or use autoscale for the vegeta clients: scale-up.sh <initial-size> <step-size> <target-size> [<sleep-duration-seconds>]
            # secs_per_point=300
            # num_steps=4
            # step_vegeta_pods=$(((${final_vegeta_pods}-${vegeta_pods_per_node})/${num_steps}))
            # ./scale-up.sh ${vegeta_pods_per_node} ${step_vegeta_pods} ${final_vegeta_pods} ${secs_per_point}

            # Wait for the pods to finish .. add another 90s on the run length to be sure
            sleepTime="$(($httpScaleRunLengthSec + 90))"
            printf "\n%s - Sleeping for ${sleepTime} seconds before collecting http-scale aggregator logs\n" "$(date +%T)"
            sleep ${sleepTime}

            # Print out aggregated metrics from the logs
            printf "\n%s - Collecting logs\n" "$(date +%T)"
            kubectl logs rc/aggregator -n "${perftest}"

            # This test runs in a cluster so it cannot write directly to Influxdb, so metrics are written to a file. Get the metrics files from the pods with the label and namespace specified.
            getInfluxdbMetricsFromPods app aggregator "${perftest}"

            # Stop cruiser metrics collection
            echo -en "${METRICS_STOP}" | nc localhost -w 5 "${metricsControlPort}"

            # Terminate cruiser metrics collection utility
            echo -en "${METRICS_TERMINATE}" | nc localhost -w 5 "${metricsControlPort}"

            # Final sleep to be sure aggregator has finished before the test is cleaned up
            sleep 90

            ./teardown.sh
            ;;
        "node-autoscaler")
            cd ${node_autoscaler_dir}

            autoscaler_name="cluster-autoscaler"
            autoscaler_Pool="asPool" # This is used as a label for the new workerpool and is used by the saler deployment yaml
            autoscaler_initial_testpod_replicas=1
            final_nodes_in_pool=2
            max_loop_wait_secs=10800 # seconds to wait for scale up or scale down before we stop : 10800 is 3 hours

            # Ensure we're starting from a clean environments
            printf "\n%s - Cleaning up old ${perftest} files\n" "$(date +%T)"
            set +e
            kubectl delete deployment scaler -n "${perftest}" --ignore-not-found=true
            set -e

            printf "\n%s - Installing ${perftest} on ${run_on_cluster}\n" "$(date +%T)"

            if [[ ${k8s_version} == *"openshift"* ]]; then
                imageTag=${k8s_major_version}.${k8s_minor_version}_openshift
            else
                imageTag=${k8s_major_version}.${k8s_minor_version}
            fi
            set -x
            # Use the first zone from the default worker pool as the zone
            autoscaler_zone=($(${perf_dir}/bin/armada-perf-client2 worker-pool get --cluster ${run_on_cluster} --worker-pool default --json | jq -r '.zones[0].id'))
            autoscaler_pool_exists=($(${perf_dir}/bin/armada-perf-client2 worker-pool ls --cluster ${run_on_cluster} --json | jq ".[] | select(.poolName==\"${autoscaler_Pool}\")" | wc -l))
            set +x

            printf "\n%s - Pool exists is: ${autoscaler_pool_exists}\n"
            if (("${autoscaler_pool_exists}" < 1)); then
                printf "\n%s - Creating autoscaler worker pool ${autoscaler_Pool}\n" "$(date +%T)"
                ${perf_dir}/bin/armada-perf-client2 worker-pool create ${cluster_type} --name ${autoscaler_Pool} --cluster ${run_on_cluster} --size-per-zone 1 --machine-type ${worker_type} --label poolLabel=${autoscaler_Pool}
                sleep 30
                set -x
                ${perf_dir}/bin/armada-perf-client2 zone add ${cluster_type} --zone ${autoscaler_zone} --worker-pool ${autoscaler_Pool} --cluster ${run_on_cluster}
                set +x
                sleep 30
                ${perf_dir}/bin/armada-perf-client2 worker-pool taint set --cluster ${run_on_cluster} --worker-pool ${autoscaler_Pool} --taint "${autoscaler_Pool}=true:NoExecute"
            else
                printf "\n%s - Reusing existing autoscaler worker pool ${autoscaler_Pool}\n" "$(date +%T)"
            fi
            # Check if the addon is already enabled.
            autoscalerAddon=$(${perf_dir}/bin/armada-perf-client2 cluster addon ls --cluster "${run_on_cluster}" --json | jq -r ".[] | select(.name==\"${autoscaler_name}\")")
            if [[ -z ${autoscalerAddon} ]]; then
                # It's not, we'll enable it
                ${perf_dir}/bin/armada-perf-client2 cluster addon enable ${autoscaler_name} --cluster "${run_on_cluster}" --version 1.0.6
            fi
	        # Wait for it to be ready
            set +e
            waitForAddonReady ${run_on_cluster} 900 ${autoscaler_name}
            if [ $? -ne 0 ]; then
                exit 1
            fi
            set -e

            autoscaler_pods_running=($(kubectl get pods -n kube-system | grep ${autoscaler_name} | grep "Running" | wc -l))
            if (("${autoscaler_pods_running}" > 0)); then
                printf "\n%s - Autoscaler pod ${autoscaler_name} is running\n"
                # Enable the Autoscaler for asPool
                as_config_map_name="iks-ca-configmap"
                kubectl get cm ${as_config_map_name} -n kube-system -o yaml > ${as_config_map_name}.yaml
                sed -i -e "s/.*{[\"]name[\"]: [\"]default[\"].*/&,\n      {\"name\": \"${autoscaler_Pool}\",\"minSize\": 1,\"maxSize\": ${autoscaler_maxnodes},\"enabled\":true}/" ${as_config_map_name}.yaml
                kubectl apply -f ${as_config_map_name}.yaml

                printf "\n\n%s - Cluster Autoscaler config map contents:\n" "$(date +%T)"
                set -x
                kubectl get cm iks-ca-configmap -n kube-system -o yaml
                set +x

                # Update image names with Carrier namespace and add the workerpool name as the nodeSelector
                rm -f scaler-deployment.yaml
                sed -e "s:%POOL-LABEL%:${autoscaler_Pool}:;s:%REPLICAS%:${autoscaler_initial_testpod_replicas}:" <template-scaler-deployment.yaml >scaler-deployment.yaml

                # Create the scaler application
                printf "\n\n%s Creating initial deployment. ${autoscaler_initial_testpod_replicas} initial replicas requested.\n" "$(date +%T)"
                kubectl create -f scaler-deployment.yaml -n "${perftest}"

                # Wait until the starting number of pods are running - this may take a while as the initial worker in the pool will be provisioning
                pods_running=($(kubectl get pods -n ${perftest} --no-headers | grep "Running" | wc -l))
                until [[ ${pods_running} -eq "${autoscaler_initial_testpod_replicas}" ]]; do
                    sleep 60
                    pods_running=($(kubectl get pods -n ${perftest} --no-headers | grep "Running" | wc -l))
                    printf "%s Creating initial deployment. ${pods_running} out of ${autoscaler_initial_testpod_replicas} requested replicas are running.\n" "$(date +%T)"
                done
                printf "\n%s Initial deployment created. ${pods_running} replicas are running.\n" "$(date +%T)"
                sleep 60

                # Run monitoring utility in the background.
                # This collects the statistics and sends the results to the metrics service
                printf "\n%s - Starting Kubernetes Event Monitoring utility\n" "$(date +%T)"
                ${perf_dir}/bin/k8s-metrics -kubecfg="${KUBECONFIG}" -namespace="${perftest}" -metrics &
                sleep 30

                # Scale up the number of replicas. Each webserver will require 1.1 cores
                printf "\n\n%s Starting scale up test - requesting ${autoscaler_testpod_replicas} replicas\n" "$(date +%T)"
                start_scale_workers_in_pool=($(kubectl get nodes --show-labels --no-headers | grep "poolLabel=${autoscaler_Pool}" | grep Ready | wc -l))

                start_scale_up_time=${SECONDS}
                kubectl scale deployment scaler --replicas=${autoscaler_testpod_replicas} -n "${perftest}"

                pods_running=($(kubectl get pods -n ${perftest} --no-headers | grep "Running" | wc -l))
                scale_up_good_run=true
                workers_autoScalerPool=0
                until [[ ${pods_running} -eq "${autoscaler_testpod_replicas}" ]]; do
                    sleep 30
                    pods_running=($(kubectl get pods -n ${perftest} --no-headers | grep "Running" | wc -l))
                    printf "%s Scaling up. ${pods_running} out of ${autoscaler_testpod_replicas} requested replicas are running.\n" "$(date +%T)"
                    workers_autoScalerPool=($(kubectl get nodes --show-labels --no-headers | grep "poolLabel=${autoscaler_Pool}" | grep Ready | wc -l))
                    ${perf_dir}/bin/send-to-bm -testname "${perftest}" -bmval "${workers_autoScalerPool}" -metricsTestName "node_scale_up_numNodes"
                    elapsed_time=$((SECONDS - start_scale_up_time))
                    if [ "${elapsed_time}" -gt "${max_loop_wait_secs}" ]; then
                        printf "\n\n%s Scale up timed out in runAuto. Timeout set to ${max_loop_wait_secs} secs. Test aborted.\n" "$(date +%T)"
                        scale_up_good_run=false
                        break
                    fi
                done

                full_scale_workers_in_pool=($(kubectl get nodes --show-labels --no-headers | grep "poolLabel=${autoscaler_Pool}" | grep Ready | wc -l))

                if [ "${scale_up_good_run}" = true ]; then
                    scale_up_time=$((SECONDS - start_scale_up_time))
                    printf "\n\n%s Scale up test complete. ${autoscaler_testpod_replicas} replicas are running. Time taken ${scale_up_time} seconds\n" "$(date +%T)"
                else
                    printf "\n\n%s Scale up test failed.\n" "$(date +%T)"
                    scale_up_time=0
                fi

                # Send to metrics service
                metricsTestNameUp=${perftest}."node_scale_up"."${start_scale_workers_in_pool}-${full_scale_workers_in_pool}"
                ${perf_dir}/bin/send-to-bm -testname "${perftest}" -bmval "${scale_up_time}" -metricsTestName "${metricsTestNameUp}"

                autoscale_wait=300
                printf "\n\n%s Scale up complete, waiting for for ${autoscale_wait} seconds before scaling down \n" "$(date +%T)"
                sleep ${autoscale_wait}

                # Scale DOWN the number of test pods to 0 (choosing zero automatically stops the k8s-metrics monitoring).
                printf "\n\n%s Starting scale down test - requesting 1 replica\n" "$(date +%T)"
                start_scale_down_time=${SECONDS}
                kubectl scale deployment scaler --replicas=1 -n "${perftest}"

                #workers_autoScalerPool=($(kubectl describe nodes | grep "poolLabel=${autoscaler_Pool}" | wc -l))
                workers_autoScalerPool=($(kubectl get nodes --show-labels --no-headers | grep "poolLabel=${autoscaler_Pool}" | grep Ready | wc -l))
                # End scaling with 2 nodes as autoscaler can leave an additional node
                scale_down_good_run=true
                until [[ "${workers_autoScalerPool}" -le "${final_nodes_in_pool}" ]]; do
                    sleep 30
                    workers_autoScalerPool=($(kubectl get nodes --show-labels --no-headers | grep "poolLabel=${autoscaler_Pool}" | grep Ready | wc -l))
                    printf "%s ${workers_autoScalerPool} nodes are running in ${autoscaler_Pool}. Scaling down to ${final_nodes_in_pool}.\n" "$(date +%T)"
                    ${perf_dir}/bin/send-to-bm -testname "${perftest}" -bmval "${workers_autoScalerPool}" -metricsTestName "node_scale_down_numNodes"
                    elapsed_time=$((SECONDS - start_scale_down_time))
                    if [ "${elapsed_time}" -gt "${max_loop_wait_secs}" ]; then
                        printf "\n\n%s Scale down timed out in runAuto. Timeout set to ${max_loop_wait_secs} secs. Test aborted.\n" "$(date +%T)"
                        scale_down_good_run=false
                        break
                    fi
                done

                if [ "${scale_down_good_run}" = true ]; then
                    scale_down_time=$((SECONDS - start_scale_down_time))
                    printf "\n\n%s Scale down test complete. ${workers_autoScalerPool} nodes are running in ${autoscaler_Pool}. Time taken = ${scale_down_time} seconds\n" "$(date +%T)"
                else
                    printf "\n\n%s Scale down test failed.\n" "$(date +%T)"
                    scale_down_time=0
                fi

                # Send to metrics service
                metricsTestNameDown=${perftest}."node_scale_down"."${full_scale_workers_in_pool}-${final_nodes_in_pool}"
                ${perf_dir}/bin/send-to-bm -testname "${perftest}" -bmval "${scale_down_time}" -metricsTestName "${metricsTestNameDown}"

                printf "\n\n%s Scale replicas to 0 to end k8s metric collection\n" "$(date +%T)"
                kubectl scale deployment scaler --replicas=0 -n "${perftest}"
                sleep 120

                # Uninstall
                printf "\n%s - Deleting test deployment\n" "$(date +%T)"
                kubectl delete deployment scaler -n ${perftest}
                sleep 30

                # Remove Addon & WorkerPool
                printf "\n%s - Removing ${autoscaler_name} Addon\n" "$(date +%T)"
                ${perf_dir}/bin/armada-perf-client2 cluster addon disable ${autoscaler_name} --cluster "${run_on_cluster}"
                sleep 60
                # Addon disable does not delete the configmap, so manually do this so we start afresh on future runs
                kubectl delete configmap -n kube-system ${as_config_map_name} 
                printf "\n%s - Removing ${autoscaler_Pool} worker pool\n" "$(date +%T)"
                ${perf_dir}/bin/armada-perf-client2 worker-pool rm --worker-pool ${autoscaler_Pool} --cluster ${run_on_cluster}
                sleep 180

            else
                printf "\n%s - Autoscaler pod ${autoscaler_name} is *NOT* running ... ${perftest} test aborted\n" "$(date +%T)"
            fi
            printf "\n\n%s - ${perftest} finished\n" "$(date +%T)"
            ;;
        "http-perf" | "https-perf")
            imageName="httpperf"
            protocol=$(echo "${perftest}" | awk -F - '{print $1}')

            cd ${http_perf_dir}

            # Get the zone and VLAN ID for LoadBalancer2 tests - classic clusters only
            if [[ "${cluster_type}" == "classic" ]]; then
                for node in $(kubectl get nodes --no-headers); do
                    node_name=$(echo ${node} | awk '{print $1}')
                    vlan=$(kubectl get node $node_name -o=jsonpath='{.metadata.labels.publicVLAN}')
                    zone=$(kubectl get node $node_name -o=jsonpath='{.metadata.labels.failure-domain\.beta\.kubernetes\.io/zone}')
                    if [[ ! -z $vlan ]]; then
                        # Found the vlan
                        echo "Using public VLAN $vlan from node $node_name for LoadBalancer2 service"
                        break
                    fi
                done
                loadBalancer2=,service.loadBalancer2.zone="${zone}",service.loadBalancer2.vlanID=\"${vlan}\"
            fi

            # Set the ingress deployment info.
            ingsize=${#ih}
            if (("${ingsize}" > 0)); then
                ingress_deployment=",ingress.hosts={${ih}},ingress.tls.hosts={${ih}},ingress.tls.secretName=${isecret}"
                if [[ ${useRoute} == true ]]; then
                    printf "\n%s - Route required for ingress host ${ih}\n" "$(date +%T)"
                    # By default, routeHostname is set to <service_name>-<namespace>.<ingress_hostname>
                    # if --hostname is not used in oc create/expose command.
                    # We set routeHostname to ${perftest}.<ingress_hostname>
                    # as http/https are using same service_name.
                    routeHostname="${perftest}.${ih}"
                    # Clients use ingress hostname ${ih} to connect to test application
                    # Setting ih to routeName for client to connect to test application via route.
                    ih="${perftest}.${ih}"
                    printf "\n%s - Route ${routeHostname} will be used\n" "$(date +%T)"
                fi
            else
                printf "%s - Ingress is unavailable.\n" "$(date +%T)"
                ingress_deployment=",ingress.enabled=false"
            fi

            if [[ ${use_edge_nodes} == false ]]; then
                num_edge_nodes=0
            fi

            original_replicas=${replicas}
            totalReplicas=$((${replicas} - ${num_edge_nodes}))
            replicas=$((${totalReplicas} / ${#zones_array[@]}))

            # Setup the zone info.
            httpZones=""
            if [[ -n ${zones} ]]; then
                httpZones=",zones={${zones}}"
            fi

            # Now deploy the application...
            printf "\n%s - Installing ${imageName} application on ${run_on_cluster}\n" "$(date +%T)"
            set -x
            ${helm_dir}/helm install "${perftest}" "${http_perf_dir}/imageDeploy/${imageName}" --namespace "${perftest}" --set clusterType="${cluster_type}" --set metricsPrefix="${METRICS_PREFIX}" --set k8sVersion="${K8S_SERVER_VERSION}" --set image.name="armada_performance_${ns_suffix}/${imageName}",replicaCount="${replicas}${httpZones}${ingress_deployment}"${loadBalancer2}
            set +x
            # Need to reset this, otherwise if running http-perf followed by https-perf replicas will be wrong.
            replicas=${original_replicas}
            # Wait to ensure application is fully up-and-running and listening for requests
            printf "\n%s - Waiting 3 mins to ensure \"${perftest}\" application is started\n" "$(date +%T)"
            sleep 180

            # Perform ingress setup on IKS clusters for Kubernetes Ingress
            if [[ ${isROKS4Plus} != true ]]; then
                if [[ -n ${isecret} ]]; then
                    # Copy the Ingress secret to the perftest namespace on IKS clusters. Needed by the Kubernetes Ingress Controller.
                    kubectl get secret ${isecret} -n default -o yaml | sed 's/default/'${perftest}'/g' | kubectl -n ${perftest} create -f -
                    kubectl annotate ing -n ${perftest} httpperf-ingress nginx.ingress.kubernetes.io/ssl-redirect="false"
                    kubectl annotate ing -n ${perftest} httpperf-ingress kubernetes.io/ingress.class=public-iks-k8s-nginx
                fi
            fi

            # Use edge nodes for running ALB pods, and the remaining nodes for the httpperf application
            useEdgeNodesForNetworkTraffic ${num_edge_nodes}
            kubectl get nodes -L dedicated

            sleep 60

            printf "\n%s - Running JMeter tests against %s\n\n" "$(date +%T)" "${k8s_cfg[0]}"

            app_kubeconfig=${KUBECONFIG}

            declare -A testPorts
            # Node Port
            testPorts["http-nodeport"]=30079
            testPorts["https-nodeport"]=30042

            # Classic Network Load Balancer 1.0
            testPorts["http-loadbalancer"]=30080
            testPorts["https-loadbalancer"]=30043

            # Classic Network Load Balancer 2.0
            testPorts["http-loadbalancer2"]=30090
            testPorts["https-loadbalancer2"]=30053

            # VPC Application Load Balancer
            testPorts["http-vpcApplicationLoadBalancer"]=30100
            testPorts["https-vpcApplicationLoadBalancer"]=30063

            # VPC Network Load Balancer
            testPorts["http-vpcNetworkLoadBalancer"]=30110
            testPorts["https-vpcNetworkLoadBalancer"]=30073

            # Ingress
            testPorts["http-ingress"]=80
            testPorts["https-ingress"]=443

            # Reset jmeterError=0 to monitor jmeter error from sendJmeterResultsReturnError called in httperfJMeterRun
            jmeterError=0

            if [[ ${useRoute} == true ]]; then
                # Create route for ingress service
                # Both http-perf and https-perf has same service name httpperf-ingress-service
                serviceName="httpperf-ingress-service"
                printf "\n%s -Creating route ${routeHostname}\n" "$(date +%T)"
                if [[ ${protocol} == "https" ]]; then
                    oc create route edge --namespace "${perftest}" --service=${serviceName} --hostname="${routeHostname}"
                else
                    # http
                    oc expose --namespace "${perftest}" svc/"${serviceName}" --hostname="${routeHostname}"
                fi
                oc get route --namespace "${perftest}"
            fi

            if [[ ${use_distributed_load_driver} == true ]]; then
                load_kubeconfig=${load_conf_yaml}
                hostsCSVFile=clusters.csv

                # Node Port
                # Output for debug purposes
                export KUBECONFIG=${app_kubeconfig}
                kubectl get pods -n "${perftest}" -o wide
                kubectl describe service -n "${perftest}" httpperf-np-service
                testPort=${testPorts["${protocol}-nodeport"]}
                generateHTTPPerfCSV nodeport distributed ${protocol} ${testPort}

                export KUBECONFIG=${load_kubeconfig}
                # Use -m standalone as default mode gets limited at around 80K req/sec
                ${jmeter_dist_dir}/bin/install.sh -m standalone -t 5 -r 250 -d 300 -s 30 -e "${ns_suffix}" "${imageName}"
                sleep 30
                httperfJMeterRun NodePort ${protocol} ${testPort} 10 20 10 250 distributed
                # Dump logs before uninstalling
                printf "\n%s - Dumping logs from jmeter-dist master \n" "$(date +%T)"
                kubectl logs -l app=jmeter-master --tail=10000
                ${helm_dir}/helm uninstall jmeter-standalone
                sleep 30

                # Classic Load Balancers
                if [[ "${cluster_type}" == "classic" ]]; then
                    # Load Balancer
                    export KUBECONFIG=${app_kubeconfig}
                    kubectl get pods -n "${perftest}" -o wide
                    kubectl describe service -n "${perftest}" httpperf-lb-service
                    testPort=${testPorts["${protocol}-loadbalancer"]}
                    generateHTTPPerfCSV loadbalancer distributed ${protocol} ${testPort}

                    export KUBECONFIG=${load_kubeconfig}
                    ${jmeter_dist_dir}/bin/install.sh -m standalone -t 5 -r 250 -d 300 -s 30 -e "${ns_suffix}" "${imageName}"
                    sleep 30
                    httperfJMeterRun LoadBalancer ${protocol} ${testPort} 10 20 10 250 distributed
                    # Dump logs before uninstalling
                    printf "\n%s - Dumping logs from jmeter-dist master \n" "$(date +%T)"
                    kubectl logs -l app=jmeter-master --tail=10000
                    ${helm_dir}/helm uninstall jmeter-standalone
                    sleep 30

                    # Load Balancer 2
                    testPort=${testPorts["${protocol}-loadbalancer2"]}
                    generateHTTPPerfCSV loadbalancer2 distributed ${protocol} ${testPort}
                    export KUBECONFIG=${load_kubeconfig}
                    ${jmeter_dist_dir}/bin/install.sh -m standalone -t 5 -r 250 -d 300 -s 30 -e "${ns_suffix}" "${imageName}"
                    sleep 30
                    httperfJMeterRun LoadBalancer2 ${protocol} ${testPort} 10 20 10 250 distributed
                    # Dump logs before uninstalling
                    printf "\n%s - Dumping logs from jmeter-dist master \n" "$(date +%T)"
                    kubectl logs -l app=jmeter-master --tail=10000
                    ${helm_dir}/helm uninstall jmeter-standalone
                    sleep 30
                fi

                # VPC Load Balancers
                if [[ "${cluster_type}" == "vpc-gen2" ]]; then
                    # VPC Application Load Balancer
                    export KUBECONFIG=${app_kubeconfig}
                    kubectl describe service -n "${perftest}" httpperf-vpc-alb-service
                    testPort=${testPorts["${protocol}-vpcApplicationLoadBalancer"]}
                    generateHTTPPerfCSV vpcApplicationLoadBalancer distributed ${protocol} ${testPort}

                    export KUBECONFIG=${load_kubeconfig}
                    ${jmeter_dist_dir}/bin/install.sh -m standalone -t 5 -r 250 -d 300 -s 30 -e "${ns_suffix}" "${imageName}"
                    sleep 30
                    httperfJMeterRun vpcApplicationLoadBalancer ${protocol} ${testPort} 10 20 10 250 distributed
                    # Dump logs before uninstalling
                    printf "\n%s - Dumping logs from jmeter-dist master \n" "$(date +%T)"
                    kubectl logs -l app=jmeter-master --tail=10000
                    ${helm_dir}/helm uninstall jmeter-standalone
                    sleep 30

                    # VPC Network Load Balancer - supported on IKS 1.19 and above, or ROKS 4.6 and above
                    if [[ (${K8S_MAJOR_VERSION} -eq 1 && ${K8S_MINOR_VERSION} -ge 19) || (${K8S_MAJOR_VERSION} -eq 4 && ${K8S_MINOR_VERSION} -ge 6) ]]; then
                        export KUBECONFIG=${app_kubeconfig}
                        kubectl describe service -n "${perftest}" httpperf-vpc-nlb-service
                        testPort=${testPorts["${protocol}-vpcNetworkLoadBalancer"]}
                        generateHTTPPerfCSV vpcNetworkLoadBalancer distributed ${protocol} ${testPort}

                        export KUBECONFIG=${load_kubeconfig}
                        ${jmeter_dist_dir}/bin/install.sh -m standalone -t 5 -r 250 -d 300 -s 30 -e "${ns_suffix}" "${imageName}"
                        sleep 30
                        httperfJMeterRun vpcNetworkLoadBalancer ${protocol} ${testPort} 10 20 10 250 distributed
                        # Dump logs before uninstalling
                        printf "\n%s - Dumping logs from jmeter-dist master \n" "$(date +%T)"
                        kubectl logs -l app=jmeter-master --tail=10000
                        ${helm_dir}/helm uninstall jmeter-standalone
                        sleep 30
                    fi
                fi

                # Ingress
                export KUBECONFIG=${app_kubeconfig}
                kubectl get pods -n "${perftest}" -o wide
                kubectl describe service -n "${perftest}" httpperf-ingress-service
                if [[ ${useRoute} == true ]]; then
                    oc get route --namespace "${perftest}"
                elif [[ ${k8s_version} == "3.11"*"openshift" ]]; then
                    kubectl get ingress httpperf-ingress -n "${perftest}" -o=yaml
                else
                    kubectl describe ingress httpperf-ingress -n "${perftest}"
                fi
                testPort=${testPorts["${protocol}-ingress"]}
                generateHTTPPerfCSV ingress distributed ${protocol} ${testPort}

                export KUBECONFIG=${load_kubeconfig}
                ${jmeter_dist_dir}/bin/install.sh -m standalone -t 5 -r 250 -d 300 -s 30 -e "${ns_suffix}" "${imageName}"
                sleep 30
                httperfJMeterRun Ingress ${protocol} ${testPort} 10 20 10 250 distributed
                # Dump logs before uninstalling
                printf "\n%s - Dumping logs from jmeter-dist master \n" "$(date +%T)"
                kubectl logs -l app=jmeter-master --tail=10000
                ${helm_dir}/helm uninstall jmeter-standalone
                sleep 30

            else
                jmeterCoreParameters="${jmeter_bin_dir}/jmeter -n -t httpperf-testplan.jmx  -JRUNLENSEC=300 -JRAMPUPSECS=0 -f"
                hostsCSVFile=hosts.csv

                # httperfJMeterRun function arguments: testType, protocol, port, startNumJMeterThreads, stopNumJMeterThreads, stepNumJMeterThreads e.g. LoadBlancer http 30079 100 300 100

                # Get Public ip node ports for httpperf, write them to the hosts.csv file and run JMeter tests.
                generateHTTPPerfCSV nodeport standalone
                httperfJMeterRun NodePort ${protocol} ${testPorts["${protocol}-nodeport"]} 100 400 100 200 standalone

                if [[ "${cluster_type}" == "classic" ]]; then
                    # Get loadbalancer ip for httpperf, write them to the hosts.csv file and run JMeter.
                    generateHTTPPerfCSV loadbalancer standalone
                    httperfJMeterRun LoadBalancer ${protocol} ${testPorts["${protocol}-loadbalancer"]} 100 400 100 200 standalone

                    # Get loadbalancer2 ip for httpperf, write them to the hosts.csv file and run JMeter.
                    generateHTTPPerfCSV loadbalancer2 standalone
                    httperfJMeterRun LoadBalancer2 ${protocol} ${testPorts["${protocol}-loadbalancer2"]} 100 400 100 200 standalone
                fi

                if [[ "${cluster_type}" == "vpc-gen2" ]]; then
                    # Get VPC application loadbalancer ip for httpperf, write them to the hosts.csv file and run JMeter.
                    generateHTTPPerfCSV vpcApplicationLoadBalancer standalone
                    httperfJMeterRun vpcApplicationLoadBalancer ${protocol} ${testPorts["${protocol}-vpcApplicationLoadBalancer"]} 100 400 100 200 standalone

                    # Get VPC network loadbalancer ip for httpperf, write them to the hosts.csv file and run JMeter.
                    generateHTTPPerfCSV vpcNetworkLoadBalancer standalone
                    httperfJMeterRun vpcNetworkLoadBalancer ${protocol} ${testPorts["${protocol}-vpcNetworkLoadBalancer"]} 100 400 100 200 standalone
                fi

                # Get ingress for httpperf, write them to the hosts.csv file and run JMeter.
                generateHTTPPerfCSV ingress standalone
                httperfJMeterRun Ingress ${protocol} ${testPorts["${protocol}-ingress"]} 100 400 100 200 standalone
            fi

            if [[ $jmeterError > 0 ]]; then
                printf "\n%s - Jmeter error found.  Failing test for investigation.\n" "$(date +%T)"
                exit 1
            fi

            # Uninstall
            export KUBECONFIG=${app_kubeconfig}
            if [[ ${useRoute} == true ]]; then
                oc delete route --namespace "${perftest}" "${serviceName}"
            fi
            ${helm_dir}/helm uninstall "${perftest}" --namespace "${perftest}"

            useEdgeNodesForNetworkTraffic 0
            ;;
        "pod-scaling")
            printf "\n%s - ${perftest} tests\n" "$(date +%T)"

            # Use a simple nginx application for our pod scaling tests
            release_name="${perftest}-nginx"

            set +e
            ${helm_dir}/helm uninstall "${release_name}" --namespace "${perftest}" 2>/dev/null
            set -e
            ${helm_dir}/helm install "${release_name}" ${pod_scaling_dir}/imageDeploy/pod-scaling --namespace "${perftest}" --set replicaCount="${initial_replicas}"

            # Wait to ensure application is fully up-and-running and listening for requests
            printf "\n%s - Waiting 40s to ensure \"${perftest}\" application is started\n" "$(date +%T)"
            sleep 40

            initial_replicas=1

            # The diferent pods per node to run - 20 is for warm-up
            # OpenShift 4 cannot fit 100 pods/node
            declare -a ppn_list=("20" "50" "80")

            cruiserMetricsController "5s" "60s" "${perftest}" "namespace"

            # Start collecting cruiser metrics
            echo -en "${METRICS_START}" | nc localhost -w 5 "${metricsControlPort}"

            # Run monitoring utility in the background.
            # This collects the statistics and sends the results to the metrics service
            printf "\n%s - Starting Kubernetes Event Monitoring utility\n" "$(date +%T)"
            ${perf_dir}/bin/k8s-metrics -kubecfg="${KUBECONFIG}" -namespace="${perftest}" -metrics &
            sleep 30

            for ppn in "${ppn_list[@]}"; do
                scale_up=$((${workers} * ${ppn} + ${initial_replicas}))
                scale_down=${initial_replicas}

                # Run scale up tests and wait for completion
                printf "\n%s - Scaling up to %d pods\n" "$(date +%T)" "${scale_up}"
                kubectl scale deployment.apps "${release_name}" -n "${perftest}" --replicas "${scale_up}"
                pods_running=($(kubectl get pods -n ${perftest} | grep "Running" | wc -l))
                until [[ ${pods_running} -eq "${scale_up}" ]]; do
                    sleep 30
                    pods_running=($(kubectl get pods -n ${perftest} --no-headers | grep "Running" | wc -l))
                done

                sleep 60

                # Run scale down tests and wait for completion
                printf "\n%s - Scaling down to %d pod(s)\n" "$(date +%T)" "${scale_down}"
                kubectl scale deployment.apps "${release_name}" -n "${perftest}" --replicas "${scale_down}"
                pod_count=($(kubectl get pods -n ${perftest} --no-headers | wc -l))
                until [[ ${pod_count} -eq "${scale_down}" ]]; do
                    sleep 30
                    pod_count=($(kubectl get pods -n ${perftest} --no-headers | wc -l))
                done

                sleep 30
            done

            # Indicate tests are completed
            sleep 15

            # Stop cruiser metrics collection
            echo -en "${METRICS_STOP}" | nc localhost -w 5 "${metricsControlPort}"

            # Terminate cruiser metrics collection utility
            echo -en "${METRICS_TERMINATE}" | nc localhost -w 5 "${metricsControlPort}"

            kubectl scale deployment.apps "${release_name}" -n "${perftest}" --replicas 0
            sleep 15

            ${helm_dir}/helm uninstall "${release_name}" --namespace "${perftest}"
            ;;
        "persistent-storage")
            allTestsRun=true
            if [[ "${cluster_type}" == "classic" ]]; then
                # https://test.cloud.ibm.com/docs/containers?topic=containers-block_storage
                declare -a scs=("ibmc-file-gold" "ibmc-block-gold") # 10 IOPS / GB

                scb=$(kubectl get storageclasses | grep block | wc -l)
                if (("${scb}" == 0)); then
                    # Need to ensure block storage plugin is installed on cluster
                    ${helm_dir}/helm repo add iks-charts https://icr.io/helm/iks-charts
                    ${helm_dir}/helm repo update
                    ${helm_dir}/helm install ibmcloud-block-storage-plugin iks-charts/ibmcloud-block-storage-plugin
                fi
                # wait for install or sometimes the next helm install fails
                sleep 60
                # Display the plugin version
                printf "\n%s - Version of block storage plugin being used\n" "$(date +%T)"
                kubectl describe deploy -n kube-system ibmcloud-block-storage-plugin | grep Image
            else
                # Enable necessary addons
                # https://test.cloud.ibm.com/docs/containers?topic=containers-vpc-block
                # https://test.cloud.ibm.com/docs/containers?topic=containers-storage-file-vpc-install
                declare -a addons=("vpc-block-csi-driver ibmc-vpc-block-10iops-tier" "vpc-file-csi-driver ibmc-vpc-file-10iops-tier")
                declare -a scs=()
                for addon in "${addons[@]}"; do
                    # Get access to addon name and storage class by converting to an array
                    addonArray=($addon)

                    # Check if the addon is already enabled.
                    vpcAddon=$(${perf_dir}/bin/armada-perf-client2 cluster addon ls --cluster "${run_on_cluster}" --json | jq -r ".[] | select(.name==\"${addonArray[0]}\")")
                    if [[ -z ${vpcAddon} ]]; then
                        # It's not, we'll enable it
                        ${perf_dir}/bin/armada-perf-client2 cluster addon enable ${addonArray[0]} --cluster "${run_on_cluster}"
                    fi

                    # Wait for it to be ready
                    set +e
                    waitForAddonReady ${run_on_cluster} 900 ${addonArray[0]}
                    if [ $? -eq 0 ]; then
                        # Addon became ready so add storage class to our test array
                        scs+=(${addonArray[1]})
                    else
                        # Skipping test so output to the log
                        printf "\n%s - Addon not available in time. Skipping test for: ${addonArray[0]}\n" "$(date +%T)"

                        # Track that this run is only a partial success
                        allTestsRun=false
                    fi
                    set -e
                done
            fi

            psvf="${cluster_config_dir}"/persistent-storage
            mkdir -p "${psvf}"

            # Now install IBM Cloud Object Storage plugin
            kubectl delete secret cos-write-access -n "${perftest}" --ignore-not-found
            kubectl create secret generic cos-write-access -n "${perftest}" --type=ibm/ibmc-s3fs --from-literal=access-key=${PROD_US_SOUTH_ARMPERF_COS_ACCESSKEYID} --from-literal=secret-key=${PROD_US_SOUTH_ARMPERF_COS_SECRETACCESSKEY}

            scos=$(kubectl get storageclasses | grep s3fs | wc -l)
            if (("${scos}" == 0)); then
                ${helm_dir}/helm repo add ibm-helm https://raw.githubusercontent.com/IBM/charts/master/repo/ibm-helm
                ${helm_dir}/helm repo update

                set +e
                ${helm_dir}/helm plugin uninstall ibmc
                set -e

                ${helm_dir}/helm pull --untar ibm-helm/ibm-object-storage-plugin --untardir "${psvf}"
                ${helm_dir}/helm plugin install ${psvf}/ibm-object-storage-plugin/helm-ibmc
                chmod a+x $HOME/.local/share/helm/plugins/helm-ibmc/ibmc.sh
                ${helm_dir}/helm ibmc install ibm-object-storage-plugin ${psvf}/ibm-object-storage-plugin --set license=true
            fi

            # Add the Cloud Object Storage test
            scs+=("ibmc-s3fs-standard-regional")

            cruiserMetricsController "30s" "60s" "${perftest}"
            success=0
            for sc in "${scs[@]}"; do
                cd ${persistent_storage_dir}/bin
                ./perf-persistent-storage.sh -v -n "${perftest}" -m -t "${perftest}" -g "${armada_registry}" -e "${ns_suffix}" -c "${sc}" -s "4000Gi" -j "/tmp/fiojobfile"

                # Send start metrics gathering control byte
                echo -en "${METRICS_START}" | nc localhost -w 5 "${metricsControlPort}"

                # Wait for job completion (max test time 3 hours)

                printf "\n%s - %d. Waiting for completion of ${perftest} test\n" "$(date +%T)" "${counter}"

                # This test runs in a cluster so it cannot write directly to Influxdb, so metrics 
                # are written to a file. Get the metrics files from the job with the name and 
                # namespace specified. Only wait for the results for the specified number of minutes. 
                rm -rf ${influxdb_metrics_dir}/${perftest}*.json 2>/dev/null
                sleep 240
                # Wait for test results
                getInfluxdbMetricsFromJob "${perftest}-job" "${perftest}" 120

                printf "\n ----------- Historic logs from job ----------- \n"
                kubectl logs job/"${perftest}-job" -n "${perftest}"
                printf "\n ---------------------------------------------- \n\n"

                # Send stop metrics gathering control byte
                echo -en "${METRICS_STOP}" | nc localhost -w 5 "${metricsControlPort}"

                # https://github.ibm.com/alchemy-containers/armada-performance/issues/789
                # Keep a record of our persistent storage volume id so that we can identify it later within Softlayer if necessary
                IFS='-' read -r -a plugin_data <<<"${sc}"
                case "${plugin_data[1]}" in
                "file")
                    pv_volume_id=$(kubectl get pv -n persistent-storage -o json | jq -r '.items[].metadata.labels.volumeId')
                    ;;
                "block")
                    pv_volume_id=$(kubectl get pv -n persistent-storage -o json | jq -r '.items[].spec.flexVolume.options.VolumeID')
                    ;;
                esac

                touch "${psvf}/${sc}-${pv_volume_id}"

                ${helm_dir}/helm uninstall "${perftest}" --namespace "${perftest}"
            done

            # Fail if not all tests were run to avoid partial failures being missed
            if [ "${allTestsRun}" != true ]; then
                printf "Some tests not attempted. Failing run"
                exit 1
            fi

            # Send terminate metrics gathering control byte
            echo -en "${METRICS_TERMINATE}" | nc localhost -w 5 "${metricsControlPort}"
            ;;
        "snapshot-storage")
            # Supported on vpc-gen2 infrastructure only
            if [[ "${cluster_type}" == "vpc-gen2" ]]; then
                vpc_block_addon_name="vpc-block-csi-driver"

                # Until the snapshot storage is available in the released driver, we need this....
                ${perf_dir}/bin/armada-perf-client2 cluster addon disable ${vpc_block_addon_name} --cluster "${run_on_cluster}"
                sleep 60
                ${perf_dir}/bin/armada-perf-client2 cluster addon enable ${vpc_block_addon_name} --version 5.0 --cluster "${run_on_cluster}"
                # Wait for it to be ready
                set +e
                waitForAddonReady ${run_on_cluster} 1800 ${vpc_block_addon_name}
                if [ $? -ne 0 ]; then
                    exit 1
                fi
                set -e

                cruiserMetricsController "5s" "60s" "${perftest}" "namespace"

                # Start collecting cruiser metrics
                echo -en "${METRICS_START}" | nc localhost -w 5 "${metricsControlPort}"

                # Run monitoring utility in the background.
                # This collects the backup/restore times and sends the results to the metrics service
                printf "\n%s - Starting VolumeSnapshot event monitoring utility\n" "$(date +%T)"
                ${perf_dir}/bin/snapshotStorage --kubecfg "${KUBECONFIG}" --namespace "${perftest}" --resource pods --resource volumeSnapshots --metrics --verbose &
                sleep 30

                cd ${snapshot_storage_dir}/bin
                ./perf-snapshot-storage.sh

                # Send stop metrics gathering control byte
                echo -en "${METRICS_STOP}" | nc localhost -w 5 "${metricsControlPort}"

                # Send terminate metrics gathering control byte
                echo -en "${METRICS_TERMINATE}" | nc localhost -w 5 "${metricsControlPort}"

                # Finally stop the event monitoring utility
                monitorPID=$(ps -ef | grep "/performance/bin/snapshotStorage" | grep -v "grep" | awk '{print $2}')
                kill ${monitorPID}
            else
                printf "\n%s - snapshot-storage test not supported on %s clusters\n" "$(date +%T)" "${cluster_type}"
                run_auto_rc=1
            fi
            ;;
        "local-storage")
            cruiserMetricsController "30s" "60s" "${perftest}"

            cd ${local_storage_dir}/bin
            ./perf-local-storage.sh -v -n "${perftest}" -m -t "${perftest}" -g "${armada_registry}" -e "${ns_suffix}" -j "/tmp/fiojobfile"

            # Send start metrics gathering control byte
            echo -en "${METRICS_START}" | nc localhost -w 5 "${metricsControlPort}"

            # Wait for job completion (max test time 3 hours)
            printf "\n%s - %d. Waiting for completion of ${perftest} test\n" "$(date +%T)" "${counter}"

            # This test runs in a cluster so it cannot write directly to Influxdb, so metrics are written to a file. Get the metrics files from the job with the name and namespace specified. Only wait for the results for the specified number of minutes.

            rm -rf ${influxdb_metrics_dir}/${perftest}*.json 2>/dev/null
            sleep 240
            # Wait for test results
            getInfluxdbMetricsFromJob "${perftest}-job" "${perftest}" 120

            printf "\n ----------- Historic logs from job ----------- \n"
            kubectl logs job/"${perftest}-job" -n "${perftest}"
            printf "\n ---------------------------------------------- \n\n"

            # Send stop metrics gathering control byte
            echo -en "${METRICS_STOP}" | nc localhost -w 5 "${metricsControlPort}"

            ${helm_dir}/helm uninstall "${perftest}" --namespace "${perftest}"

            # Send terminate metrics gathering control byte
            echo -en "${METRICS_TERMINATE}" | nc localhost -w 5 "${metricsControlPort}"
            ;;
        "odf-storage" | "odf-storage-nodisable")
            num_cores=$(echo ${worker_type} | cut -d "." -f2 | cut -d "x" -f1)
            if [[ ${num_cores} -lt 16 ]]; then
                printf "\n ODF Tests require a minimum of 16 cores, current worker_type is %s \n" "${num_cores}"
                exit 1
            fi

            if [[ "${cluster_type}" != "vpc-gen2" ]]; then
                printf "\n ODF Tests are currently only supported on vpc-gen2 clusters by the automation \n"
                exit 1
            fi

            disableODF="true"
            if [[ "${perftest}" == "odf-storage-nodisable" ]]; then
                printf "\n Test ${perftest} specified, will not disable ODF at the end of the test \n"
                disableODF="false"
                perftest="odf-storage"
                # Need to setup the namespace as the generic code will have configured it for odf-storage-nodisable namespace
                source ${armada_perf_dir}/automation/bin/setupRegistryAccess.sh "${perftest}"
                source ${armada_perf_dir}/automation/bin/enableRootUserOnOpenshift.sh "${perftest}" ${oc_insecure_login}
            fi
            # Enable ODF addon and create StorageCluster
            cd ${persistent_storage_dir}/odf
            ./enableOdf.sh ${run_on_cluster}
            # We will compare running with the standard vpc block storage (Which is used as backing storage for ODF) with the ODF Storage classes
            declare -a scs=("ibmc-vpc-block-metro-10iops-tier" "ocs-storagecluster-ceph-rbd" "ocs-storagecluster-cephfs")

            psvf="${cluster_config_dir}"/persistent-storage
            mkdir -p "${psvf}"

            cruiserMetricsController "30s" "60s" "${perftest}"
            success=0
            for sc in "${scs[@]}"; do
                cd ${persistent_storage_dir}/bin
                ./perf-persistent-storage.sh -v -n "${perftest}" -m -t "${perftest}" -g "${armada_registry}" -e "${ns_suffix}" -c "${sc}" -s "4000Gi" -j "/tmp/fiojobfile"

                # Send start metrics gathering control byte
                echo -en "${METRICS_START}" | nc localhost -w 5 "${metricsControlPort}"

                # Wait for job completion (max test time 3 hours)

                printf "\n%s - %d. Waiting for completion of ${perftest} test\n" "$(date +%T)" "${counter}"

                # This test runs in a cluster so it cannot write directly to Influxdb, so metrics are written to a file. Get the metrics files from the job with the name and namespace specified. Only wait for the results for the specified number of minutes.

                rm -rf ${influxdb_metrics_dir}/${perftest}*.json 2>/dev/null
                sleep 240
                # Wait for test results
                getInfluxdbMetricsFromJob "persistent-storage-job" "${perftest}" 120

                printf "\n ----------- Historic logs from job ----------- \n"
                kubectl logs job/persistent-storage-job -n "${perftest}"
                printf "\n ---------------------------------------------- \n\n"

                # Send stop metrics gathering control byte
                echo -en "${METRICS_STOP}" | nc localhost -w 5 "${metricsControlPort}"

                # https://github.ibm.com/alchemy-containers/armada-performance/issues/789
                # Keep a record of our persistent storage volume id so that we can identify it later within Softlayer if necessary
                IFS='-' read -r -a plugin_data <<<"${sc}"
                case "${plugin_data[1]}" in
                "file")
                    pv_volume_id=$(kubectl get pv -n persistent-storage -o json | jq -r '.items[].metadata.labels.volumeId')
                    ;;
                "block")
                    pv_volume_id=$(kubectl get pv -n persistent-storage -o json | jq -r '.items[].spec.flexVolume.options.VolumeID')
                    ;;
                esac

                touch "${psvf}/${sc}-${pv_volume_id}"

                ${helm_dir}/helm uninstall "persistent-storage" --namespace "${perftest}"
            done
            # Send terminate metrics gathering control byte
            echo -en "${METRICS_TERMINATE}" | nc localhost -w 5 "${metricsControlPort}"

            if [[ "${disableODF}" == "true" ]]; then
                printf "\n Disabling ODF \n"
                # Delete StorageCluster and disable addon
                cd ${persistent_storage_dir}/odf
                ./disableOdf.sh ${run_on_cluster}
            fi

            ;;
        "odf-storage-parallel" | "odf-storage-parallel-nodisable")
            num_cores=$(echo ${worker_type} | cut -d "." -f2 | cut -d "x" -f1)
            if [[ ${num_cores} -lt 16 ]]; then
                printf "\n ODF Tests require a minimum of 16 cores, current worker_type is %s \n" "${num_cores}"
                exit 1
            fi

            if [[ "${cluster_type}" != "vpc-gen2" ]]; then
                printf "\n ODF Tests are currently only supported on vpc-gen2 clusters by the automation \n"
                exit 1
            fi

            disableODF="true"
            if [[ "${perftest}" == "odf-storage-parallel-nodisable" ]]; then
                printf "\n Test ${perftest} specified, will not disable ODF at the end of the test \n"
                disableODF="false"
            fi

            # To help with organising things in Grafana use the same perf test name as
            # the non-parallel case. This will let us easily compare both parallel and
            # single pod runs alongside each other.
            perftest="odf-storage"

            # Need to setup the namespace as the generic code will have configured it for odf-storage-parallel/nodisable namespace
            source ${armada_perf_dir}/automation/bin/setupRegistryAccess.sh "${perftest}"
            source ${armada_perf_dir}/automation/bin/enableRootUserOnOpenshift.sh "${perftest}" ${oc_insecure_login}

            # Enable ODF addon and create StorageCluster
            cd ${persistent_storage_dir}/odf
            ./enableOdf.sh ${run_on_cluster}

            # Only test the ODF storage classes using the parallel runs. Block storage
            # comparison is done as part of the single threaded storage tests.
            declare -a scs=("ocs-storagecluster-ceph-rbd" "ocs-storagecluster-cephfs")

            # To keep test times down only do the parallel runs as part of this test
            # Comparison can again be carried out against the single threaded tests.
            declare -a podcounts=("5" "10")

            # TODO: Do we need this in the parallel case? Should we be making a
            #       note of all the pvcs that are created for these tests?
            psvf="${cluster_config_dir}"/persistent-storage
            mkdir -p "${psvf}"

            cruiserMetricsController "30s" "60s" "${perftest}"

            cd ${persistent_storage_dir}/bin

            set +e # Disable due to certain checks in perf-parallel-storage.sh

            # In the ODF case use pod affinity based on zone to ensure a better
            # distribution of our pods across all available nodes.
            affinityLabel="ibm-cloud.kubernetes.io/zone"

            # Iterate over all storage classes and podcounts to
            # generate a "complete" set of results.
            success=0
            for sc in "${scs[@]}"; do
                for podcount in "${podcounts[@]}"; do

                    # This script will run tests for a pre-defined set of block sizes and rw modes
                    # using the supplied number of pods to test them in parallel.
                    ./perf-parallel-storage.sh -v -n "${perftest}" -m -t "${perftest}" -g "${armada_registry}" -e "${ns_suffix}" -c "${sc}" -s "4000Gi" -p ${podcount} -al "${affinityLabel}" -av "${zones}"

                    # NOTE: Waiting for completion and metrics handling/upload
                    #       is done by the above script in the parallel case.

                done # podcount
            done     # storage class

            set -e # Re-enable

            if [[ "${disableODF}" == "true" ]]; then
                printf "\n Disabling ODF \n"
                # Delete StorageCluster and disable addon
                cd ${persistent_storage_dir}/odf
                ./disableOdf.sh ${run_on_cluster}
            fi

            ;;
        "portworx-storage" | "portworx-storage-nodisable")
            num_cores=$(echo ${worker_type} | cut -d "." -f2 | cut -d "x" -f1)
            if [[ ${num_cores} -lt 16 ]]; then
                printf "\n Portworx Tests require a minimum of 16 cores, current worker_type is %s \n" "${num_cores}"
                exit 1
            fi

            if [[ "${cluster_type}" != "vpc-gen2" ]]; then
                printf "\n Portworx Tests are currently only supported on vpc-gen2 clusters by the automation \n"
                exit 1
            fi

            disablePortworx="true"
            if [[ "${perftest}" == "portworx-storage-nodisable" ]]; then
                printf "\n Test ${perftest} specified, will not disable Portworx at the end of the test \n"
                disablePortworx="false"
                perftest="portworx-storage"
                # Need to setup the namespace as the generic code will have configured it for portworx-storage-nodisable namespace
                source ${armada_perf_dir}/automation/bin/setupRegistryAccess.sh "${perftest}"
                source ${armada_perf_dir}/automation/bin/enableRootUserOnOpenshift.sh "${perftest}" ${oc_insecure_login}
            fi

            # Enable Portworx
            cd ${persistent_storage_dir}/portworx
            ./enablePortworx.sh ${run_on_cluster}
            # We will compare running with the standard vpc block storage (Which is used as backing storage for Portworx) with the Portworx Storage classes
            #  "portworx-db2-sc" is available but wasn't working, so removed for now.
            declare -a scs=("portworx-null-sc" "portworx-shared-sc" "portworx-db-sc" "ibmc-vpc-block-metro-10iops-tier")

            psvf="${cluster_config_dir}"/persistent-storage
            mkdir -p "${psvf}"

            cruiserMetricsController "30s" "60s" "${perftest}"
            success=0
            for sc in "${scs[@]}"; do
                cd ${persistent_storage_dir}/bin
                # For Portworx storage classes use the stork scheduler
                if [[ "${sc}" == portworx* ]]; then
                    ./perf-persistent-storage.sh -v -n "${perftest}" -m -t "portworx-storage" -g "${armada_registry}" -e "${ns_suffix}" -c "${sc}" -s "4000Gi" -j "/tmp/fiojobfile" -x "stork"

                    # Find a Portworx pod so that we can run some pxctl commmands
                    PX_POD=$(kubectl get pods -l name=portworx -n kube-system -o jsonpath='{.items[0].metadata.name}')
                    # We might have to wait for the volume to appear before we can
                    # inspect it so loop here until we get a value
                    PX_VOLUME=""
                    while [[ -z ${PX_VOLUME} ]]; do
                        ((c++)) && ((c == 10)) && break # Only attempt 10 times
                        echo "Waiting to get Portworx volume ID..."
                        sleep 10s
                        PX_VOLUME=$(kubectl exec ${PX_POD} -n kube-system -- /opt/pwx/bin/pxctl volume list | sed -n 2p | awk '{print $1}')
                    done
                    # Only attempt if we managed to get a volume ID
                    if [[ -n ${PX_VOLUME} ]]; then
                        # Print out volume information
                        printf "\n%s - Portworx volume status: %s \n" "$(date +%T)"
                        kubectl exec ${PX_POD} -n kube-system -- /opt/pwx/bin/pxctl volume inspect ${PX_VOLUME}
                    fi
                else
                    ./perf-persistent-storage.sh -v -n "${perftest}" -m -t "portworx-storage" -g "${armada_registry}" -e "${ns_suffix}" -c "${sc}" -s "4000Gi" -j "/tmp/fiojobfile"
                fi

                # Send start metrics gathering control byte
                echo -en "${METRICS_START}" | nc localhost -w 5 "${metricsControlPort}"

                # Wait for job completion (max test time 3 hours)
                printf "\n%s - %d. Waiting for completion of ${perftest} test\n" "$(date +%T)" "${counter}"

                # This test runs in a cluster so it cannot write directly to Influxdb, so metrics are written to a file. Get the metrics files from the job with the name and namespace specified. Only wait for the results for the specified number of minutes.
                rm -rf ${influxdb_metrics_dir}/${perftest}*.json 2>/dev/null
                sleep 240
                # Wait for test results
                getInfluxdbMetricsFromJob "persistent-storage-job" "${perftest}" 120

                printf "\n ----------- Historic logs from job ----------- \n"
                kubectl logs job/persistent-storage-job -n "${perftest}"
                printf "\n ---------------------------------------------- \n\n"

                # Send stop metrics gathering control byte
                echo -en "${METRICS_STOP}" | nc localhost -w 5 "${metricsControlPort}"

                # https://github.ibm.com/alchemy-containers/armada-performance/issues/789
                # Keep a record of our persistent storage volume id so that we can identify it later within Softlayer if necessary
                IFS='-' read -r -a plugin_data <<<"${sc}"
                case "${plugin_data[1]}" in
                "file")
                    pv_volume_id=$(kubectl get pv -n persistent-storage -o json | jq -r '.items[].metadata.labels.volumeId')
                    ;;
                "block")
                    pv_volume_id=$(kubectl get pv -n persistent-storage -o json | jq -r '.items[].spec.flexVolume.options.VolumeID')
                    ;;
                esac

                touch "${psvf}/${sc}-${pv_volume_id}"

                ${helm_dir}/helm uninstall "persistent-storage" --namespace "${perftest}"
            done
            # Send terminate metrics gathering control byte
            echo -en "${METRICS_TERMINATE}" | nc localhost -w 5 "${metricsControlPort}"
            if [[ "${disablePortworx}" == "true" ]]; then
                printf "\n Disabling Portworx \n"
                # Remove Portworx from cluster
                cd ${persistent_storage_dir}/portworx
                ./disablePortworx.sh ${run_on_cluster}
            fi

            ;;
        "portworx-storage-parallel" | "portworx-storage-parallel-nodisable")
            num_cores=$(echo ${worker_type} | cut -d "." -f2 | cut -d "x" -f1)
            if [[ ${num_cores} -lt 16 ]]; then
                printf "\n Portworx Tests require a minimum of 16 cores, current worker_type is %s \n" "${num_cores}"
                exit 1
            fi

            if [[ "${cluster_type}" != "vpc-gen2" ]]; then
                printf "\n Portworx Tests are currently only supported on vpc-gen2 clusters by the automation \n"
                exit 1
            fi

            disablePortworx="true"
            if [[ "${perftest}" == "portworx-storage-parallel-nodisable" ]]; then
                printf "\n Test ${perftest} specified, will not disable Portworx at the end of the test \n"
                disablePortworx="false"
            fi

            # To help with organising things in Grafana use the same perf test name as
            # the non-parallel case. This will let us easily compare both parallel and
            # single pod runs alongside each other.
            perftest="portworx-storage"

            # Need to setup the namespace as the generic code will have configured it
            # for a portworx-storage-parallel/nodisable namespace.
            source ${armada_perf_dir}/automation/bin/setupRegistryAccess.sh "${perftest}"
            source ${armada_perf_dir}/automation/bin/enableRootUserOnOpenshift.sh "${perftest}" ${oc_insecure_login}

            # Enable Portworx
            cd ${persistent_storage_dir}/portworx
            ./enablePortworx.sh ${run_on_cluster}

            # Only test the Portworx storage classes using the parallel runs. Block storage
            # comparison is done as part of the single threaded storage tests.
            declare -a scs=("portworx-null-sc" "portworx-shared-sc" "portworx-db-sc")

            # To keep test times down only do the parallel runs as part of this test
            # Comparison can again be carried out against the single threaded tests.
            declare -a podcounts=("5" "10")

            psvf="${cluster_config_dir}"/persistent-storage
            mkdir -p "${psvf}"

            cruiserMetricsController "30s" "60s" "${perftest}"

            cd ${persistent_storage_dir}/bin

            set +e # Disable due to certain checks in perf-parallel-storage.sh

            # Iterate over all storage classes and podcounts to
            # generate a "complete" set of results.
            success=0
            for sc in "${scs[@]}"; do
                for podcount in "${podcounts[@]}"; do

                    # NOTE: Always use the Stork scheduler for Portworx storage classes. Because
                    #       of this we don't supply any label/values for the pod affinity logic.
                    #
                    # This script will run tests for a pre-defined set of block sizes and rw modes
                    # using the supplied number of pods to test them in parallel.
                    ./perf-parallel-storage.sh -v -n "${perftest}" -m -t "${perftest}" -g "${armada_registry}" -e "${ns_suffix}" -c "${sc}" -s "4000Gi" -p ${podcount} -x "stork"

                    # NOTE: Waiting for completion and metrics handling/upload
                    #       is done by the above script in the parallel case.

                done # podcount
            done     # storage class

            set -e # Re-enable

            if [[ "${disablePortworx}" == "true" ]]; then
                printf "\n Disabling Portworx \n"
                # Remove Portworx from cluster
                cd ${persistent_storage_dir}/portworx
                ./disablePortworx.sh ${run_on_cluster}
            fi

            ;;
        "acmeair" | "acmeair-istio" | "acmeair-istio-extras" | "acmeair-image" | "acmeair-istio-image")
            metricsTestName="${perftest}"."nodes${workers}"
            cd "${GOPATH}"/src/blueperf/helper
            cluster_name="${run_on_cluster}"

            isIstio=false
            istio=""
            istioExtras=""
            imageFlag=""

            if [[ "${ingress_ok}" == false ]]; then
                echo
                echo Ingress hostname or secret is empty. Stopping test.
                echo
                exit 1
            fi

            if [[ "${perftest}" == *"-image" ]]; then
                imageFlag="--image"
            fi

            if [[ ${useRoute} == true ]]; then
                ih="acmeair.${ih}"
            fi

            echo isIstio: $isIstio
            echo istio: $istio
            echo istioExtras: $istioExtras
            echo ih: $ih
            echo imageFlag: $imageFlag

            #Export api key for use by the called scripts
            export API_KEY=${ARMADA_PERFORMANCE_API_KEY}

            if [[ "${existing_cluster}" == true ]]; then
                # It may be a clean cluster and nothing to undeploy
                echo "Cleaning up acmeair resources from last test"
                echo "Ignore clean up errors as resources may not exists"
                set +e
                # Do login and undeploy to get back to clean start
                ./IBMCloud_KS.sh -l ${istio} ${routeParameter} -cl "${cluster_name}"
                ./IBMCloud_KS.sh -u ${istio} ${routeParameter} -cl "${cluster_name}"
                istio/istio_config.sh uninstall "${cluster_name}" "${istioExtras}"
                # Last test may use route or ingress, remove all
                deleteRoute acmeair
                deleteIngress acmeair
                set -e
                echo "Cleaning up acmeair resources from last test completed"
            fi

            if [[ "${perftest}" == "acmeair-istio"* ]]; then
                # Enable istio addon
                isIstio=true
                istio="--istio"

                if [[ "${perftest}" == "acmeair-istio-extras" ]]; then
                    isIstioExtras=true
                    istioExtras="--extras"
                fi
                # Generate acmeair-ingress.yaml from template file before calling istio_config.sh
                sed "s#INGRESS_HOST#${ih}#g" istio/acmeair-ingress-template.yaml | sed "s#INGRESS_SECRET#${isecret}#g" >istio/acmeair-ingress.yaml
                istio/istio_config.sh install "${cluster_name}" "${K8S_SERVER_VERSION}" "${istioExtras}"
                rm istio/acmeair-ingress.yaml
                # Now change ingress for Istio
                ih="acmeair.${ih}"
                # Copy the Ingress secret to the istio-system namespace for istio tests. Needed by the Kubernetes Ingress Controller.
                kubectl get secret ${isecret} -n default -o yaml | sed 's/default/istio-system/g' | kubectl -n istio-system create -f -
            fi

            # Add Acmeair labels to nodes for Acmeair pods
            scripts/label_nodes.sh

            # install the prereq clis's if needed
            ./IBMCloud_KS.sh -p ${istio} ${routeParameter} -cl "${cluster_name}"
            # deploy and populate the Acmeair databases
            ./IBMCloud_KS.sh -a ${istio} ${routeParameter} ${imageFlag} -cl "${cluster_name}"

            kubectl get pod -o wide

            if [[ ${isIstio} == false ]]; then
                # Label nodes and re-distribute acmeair pods.  Script will list nodes and pods after change
                # Istio version don't need to patch as istio provides own deployment in istio directory
                # in this GIT already with labels added
                ${GOPATH}/src/blueperf/helper/scripts/patch_deployment.sh
            fi

            # Metrics collection parameters : delay and interval between measurements
            cruiserMetricsController "30s" "60s" "${perftest}"

            # Start cruiser metrics collection
            echo -en "${METRICS_START}" | nc localhost -w 5 "${metricsControlPort}"

            printf "Now run jmeter tests...\n"

            cd ${perf_dir}/src/blueperf/acmeair-driver/acmeair-jmeter/scripts

            printf "Remove any old results\n"
            rm -f results*.jtl
            rm -f acmeair.log
            rm -f output*.csv

            jmeterJmxFile="AcmeAir-microservices-mpJwt.jmx"

            # Reset jmeterError=0 to monitor jmeter error before calling sendJmeterResultsReturnError looping through threads
            jmeterError=0

            printf "Run warm-up with 1 thread\n"
            ${jmeter_bin_dir}/jmeter -n -t ${jmeterJmxFile} -DCookieManager.save.cookies=true -DusePureIDs=true -JHOST="${ih}" -JPORT=80 -j acmeair.log -JTHREAD=1 -JUSER=999 -JDURATION=60 -JRAMP=60 -f -l results1.jtl
            printf "Now run full test with various number of threads\n"
            for jthreads in 20 60; do
                resFile=results"${jthreads}".jtl
                resCSVFile=output"${jthreads}".csv
                # Clear the booking database before each run
                printf "Clearing booking DB at ${ih}\n"
                curl http://${ih}/booking/loader/load || true

                printf "Running with ${jthreads} threads\n"
                ${jmeter_bin_dir}/jmeter -n -t ${jmeterJmxFile} -DCookieManager.save.cookies=true -DusePureIDs=true -JHOST="${ih}" -JPORT=80 -j acmeair.log -JTHREAD="${jthreads}" -JUSER=999 -JDURATION=300 -JRAMP=60 -f -l "${resFile}"

                # Convert output to a CSV
                ${jmeter_bin_dir}/JMeterPluginsCMD.sh --tool Reporter --plugin-type AggregateReport --input-jtl "${resFile}" --generate-csv "${resCSVFile}"
                cat ${resCSVFile}

                # Send to metrics service
                sendJmeterResultsReturnError -testname "${perftest}" -numThreads "${jthreads}" -resultsFile "${resCSVFile}" -metricsTestName "${metricsTestName}" -metrics
            done

            # Stop cruiser metrics collection
            echo -en "${METRICS_STOP}" | nc localhost -w 5 "${metricsControlPort}"

            # Terminate cruiser metrics collection utility
            echo -en "${METRICS_TERMINATE}" | nc localhost -w 5 "${metricsControlPort}"

            if [[ $jmeterError > 0 ]]; then
                printf "\n%s - Jmeeter error found.  Failing test.\n" "$(date +%T)"
                exit 1
            fi

            cd "${GOPATH}"/src/blueperf/helper

            # Undeploy acmeair and delete deployment to clean up pods
            ./IBMCloud_KS.sh -u ${istio} ${routeParameter} -cl "${cluster_name}"
            if [[ ${useRoute} == true ]]; then
                deleteRoute acmeair
            else
                deleteIngress acmeair
            fi
            # Remove Acmeair labels from nodes
            scripts/unlabel_nodes.sh

            if [[ ${isIstio} == true ]]; then
                istio/istio_config.sh uninstall "${cluster_name}" "${istioExtras}"
            fi
            ;;
        "olb-java")
            metricsTestName="${perftest}"."nodes${workers}"
            cd "${GOPATH}"/src/ibmperf/olb-java

            if [[ "${ingress_ok}" == false ]]; then
                echo
                echo Ingress hostname or secret is empty. Stopping test.
                echo
                exit 1
            fi

            # Remove output files from previous runs
            rm -f results*.jtl
            rm -f olb.log
            rm -f output*.csv

            # Delete route if there
            if [[ ${useRoute} == true ]]; then
                deleteRoute olb-java
            fi

            touch hosts.csv

            # There's a bug in the helm chart which means it won't install (have notified @moss)
            # For now, just delete the problematic file, (it's not important)
            if [[ -f ./olb-java/charts/olb-stub-java/templates/NOTES.txt ]]; then
                rm -f ./olb-java/charts/olb-stub-java/templates/NOTES.txt
            fi
            set +e
            # The timeout here is quite long - we have seen issues with it taking a long time to pull the image from docker hub
            ${helm_dir}/helm install ${perftest} ${perftest} --namespace "${perftest}" --set ingress.host=olb.${ih} --set image.repository="${armada_registry}/armada_performance_${ns_suffix}/olb-java",olb-stub-java.image.repository="${armada_registry}/armada_performance_${ns_suffix}/olb-stub-java" --set image.tag=latest,olb-stub-java.image.tag=latest --wait --timeout=600s
            if [ $? -ne 0 ]; then
                echo "Helm install of olb-java timed out - printing pod status, then exiting:"
                kubectl get pods -A -o=wide
                kubectl describe pod --namespace "${perftest}"
                exit 1
            fi
            set -e

            # Perform ingress setup on IKS clusters for Kubernetes Ingress
            if [[ ${isROKS4Plus} != true ]]; then
                # Copy the Ingress secret to the perftest namespace on IKS clusters. Needed by the Kubernetes Ingress Controller.
                kubectl get secret ${isecret} -n default -o yaml | sed 's/default/'${perftest}'/g' | kubectl -n ${perftest} create -f -
                kubectl annotate ing -n ${perftest} olb-java-olb-java nginx.ingress.kubernetes.io/ssl-redirect="false"
                kubectl annotate ing -n ${perftest} olb-java-olb-java kubernetes.io/ingress.class=public-iks-k8s-nginx
            fi

            if [[ ${use_distributed_load_driver} == true ]]; then
                cp OLB.jmx ${jmeter_dist_dir}/tests/${perftest}/test.jmx

                app_kubeconfig=${KUBECONFIG}
                load_kubeconfig=${load_conf_yaml}
                export KUBECONFIG=${load_kubeconfig}
                ${jmeter_dist_dir}/bin/install.sh -s 30 -e "${ns_suffix}" -a "-GHOST=olb.${ih} -GTHREAD=100 -GRAMP=20" "${perftest}"
                sleep 30
                export KUBECONFIG=${app_kubeconfig}

                client_threads=(20 25 35)
            else
                client_threads=(400 500 600 700 800)
            fi

            if [[ ${useRoute} == true ]]; then
                # Create route for olb-java service
                serviceName="olb-java-olb-java-service"
                printf "\n%s -Creating route olb.${ih}\n" "$(date +%T)"
                oc expose --namespace "${perftest}" svc/"${serviceName}" --hostname="olb.${ih}"
                oc get route --namespace "${perftest}"
            fi

            # Metrics collection parameters : delay and interval between measurements
            cruiserMetricsController "30s" "60s" "${perftest}"

            # Start cruiser metrics collection
            echo -en "${METRICS_START}" | nc localhost -w 5 "${metricsControlPort}"

            printf "Now run olb jmeter tests...\n"

            olbWarmupThreads=200
            warmupDuration=300
            testDuration=300
            printf "Run warm-up with %s threads\n" "${olbWarmupThreads}"
            ${jmeter_bin_dir}/jmeter -n -t OLB.jmx -JHOST="olb.${ih}" -JPORT=80 -j olb.log -JTHREAD="${olbWarmupThreads}" -JDURATION=${warmupDuration} -JRAMP=60

            # Reset jmeterError=0 to monitor jmeter error before calling sendJmeterResultsReturnError in thread loops
            jmeterError=0

            printf "Now run full test with various number of threads\n"
            for jthreads in ${client_threads[@]}; do
                printf "Running with ${jthreads} threads\n"

                resCSVFile=output"${jthreads}".csv

                if [[ ${use_distributed_load_driver} == true ]]; then
                    export KUBECONFIG=${load_kubeconfig}
                    "${jmeter_dist_dir}/bin/run.sh" -t ${jthreads} -r 0 -d ${testDuration} >"${resCSVFile}"
                    export KUBECONFIG=${app_kubeconfig}
                else
                    resFile=results"${jthreads}".jtl

                    ${jmeter_bin_dir}/jmeter -n -t OLB.jmx -JHOST="olb.${ih}" -JPORT=80 -j olb.log -JTHREAD="${jthreads}" -JDURATION=${testDuration} -JRAMP=0 -f -l "${resFile}"

                    # Convert output to a CSV
                    ${jmeter_bin_dir}/JMeterPluginsCMD.sh --tool Reporter --plugin-type AggregateReport --input-jtl "${resFile}" --generate-csv "${resCSVFile}"
                fi

                # Send to metrics service
                cat ${resCSVFile}
                sendJmeterResultsReturnError -testname "${perftest}" -numThreads "${jthreads}" -resultsFile "${resCSVFile}" -metricsTestName "${metricsTestName}" -metrics
            done

            # Stop cruiser metrics collection
            echo -en "${METRICS_STOP}" | nc localhost -w 5 "${metricsControlPort}"

            # Terminate cruiser metrics collection utility
            echo -en "${METRICS_TERMINATE}" | nc localhost -w 5 "${metricsControlPort}"

            if [[ $jmeterError > 0 ]]; then
                printf "\n%s - Jmeter error found.  Failing test for investigation.\n" "$(date +%T)"
                exit 1
            fi

            # Undeploy
            if [[ ${use_distributed_load_driver} == true ]]; then
                export KUBECONFIG=${load_kubeconfig}
                # Dump logs before uninstalling
                printf "\n%s - Dumping logs from jmeter-dist master \n" "$(date +%T)"
                kubectl logs -l app=jmeter-master --tail=10000
                ${helm_dir}/helm uninstall jmeter-dist
                export KUBECONFIG=${app_kubeconfig}
            fi

            ${helm_dir}/helm uninstall ${perftest} --namespace "${perftest}"
            ;;
        "zeroworkerclusters" | "apiserver-load")
            if ((num_zero_clusters > 0)); then
                printf "\n%s - Creating zero worker cluster(s)\n" "$(date +%T)"

                kvstr=""
                if [ "${k8s_version}" != "default" ]; then
                    kvstr="--kube-version ${k8s_version} "
                    export K8S_SERVER_VERSION="${k8s_version}"
                else
                    export K8S_SERVER_VERSION=$(${perf_dir}/bin/armada-perf-client2 versions --json | jq -j '.kubernetes[] | select(.default==true) | .major, "_", .minor')
                fi

                # **** set +e and -e : TEMPORARY HACK TO WORKAROUND https://github.ibm.com/alchemy-containers/armada-api/issues/2072 ****
                # Check if all clusters already exist
                current_zero_clusters=$(${perf_dir}/bin/armada-perf-client2 cluster ls --json | jq --arg cnp "${cluster_prefix_zero_workers}" '.[] | select(.name | startswith($cnp)) | .id' | head -${num_zero_clusters} | wc -l)

                test_start_time=$(date --iso-8601=seconds)
                if ((${current_zero_clusters} != ${num_zero_clusters})); then
                    printf "\n%s - Creating Cruiser(s)\n" "$(date +%T)"
                    if [[ ${perftest} == "zeroworkerclusters" ]]; then
                        enableMetrics="--metrics"
                    else
                        enableMetrics=""
                    fi

                    if [[ ${cluster_type} == "satellite-"* ]]; then
                        infrastructure="--location ${location}"
                    else
                        infrastructure="--machine-type ${worker_type}"
                    fi

                    ${perf_dir}/bin/armada-perf-client2 cluster align --name "${cluster_prefix_zero_workers}" --quantity ${num_zero_clusters} --threads ${num_zero_clusters} --provider ${cluster_type} --workers=-1 ${infrastructure} ${kvstr} --poll-interval 30s --timeout 60m "${enableMetrics}"

                    # Esnure any historical configuration data is cleaned up
                    zw_cluster_config_dir="${perf_dir}"/config/"${cluster_prefix_zero_workers}"
                    rm -rf "${zw_cluster_config_dir}"*
                else
                    printf "\n%s - Using existing zero worker Cruisers\n" "$(date +%T)"
                fi

                # Sleep after we have created the zero worker Cluster so we get steady state Carrier metrics
                sleep_collect_carrier_metrics=300
                printf "\n%s - Sleeping for %d seconds to record idle Carrier metrics\n" "$(date +%T)" "${sleep_collect_carrier_metrics}"
                sleep ${sleep_collect_carrier_metrics}

                # Configure apiserver load testing
                printf "\n%s - Running zero worker apiserver load test\n" "$(date +%T)"
                keystorepwd=$(echo $RANDOM | md5sum | head -c 20)
                output_file=apiserver_samples

                if [[ ${perftest} == "zeroworkerclusters" ]]; then
                    jmeter_threads=150
                    metrics_testname=zeroWorkers-apiserver-load

                    cd ${k8s_apiserver_dir}
                    ${k8s_apiserver_dir}/api-loadtest-config.sh "${cluster_prefix_zero_workers}" "${keystorepwd}"
                    cd jmeter-config
                    ${jmeter_bin_dir}/jmeter -n -t K8SAPIServer-direct.jmx -JRUNLENSEC=600 -JRAMPUPSECS=0 -JTPUTMINS=6000 -JTHREAD="${jmeter_threads}" -Djavax.net.ssl.keyStore=./cert.jks -Djavax.net.ssl.keyStorePassword="${keystorepwd}" -Dhttpclient4.time_to_live=1800000 -f -l "${output_file}.out"
                    printf "\n%s - Zero worker apiserver load test completed\n" "$(date +%T)"
                    test_end_time=$(date --iso-8601=seconds)

                    # Convert output to a CSV
                    ${jmeter_bin_dir}/JMeterPluginsCMD.sh --tool Reporter --plugin-type AggregateReport --input-jtl "${output_file}.out" --generate-csv "${output_file}.csv"
                else
                    # Running in distributed mode (requires larger cluster than our standard automated tests to drive the workload on the carrier)
                    metrics_testname="k8s-apiserver-dist-load"

                    jmeter_dist_test_dir="${jmeter_dist_dir}/tests"

                    # Clean up any old test
                    jmdl_config_dir="${jmeter_dist_dir}/config"
                    if [ -d "${jmdl_config_dir}" ]; then
                        rm -r "${jmdl_config_dir}"
                    fi
                    mkdir "${jmdl_config_dir}"

                    ${k8s_apiserver_dir}/api-loadtest-config.sh "${cluster_prefix_zero_workers}" "${keystorepwd}"

                    cp ${k8s_apiserver_dir}/jmeter-config/cert.jks ${jmeter_dist_test_dir}/${perftest}
                    cp ${k8s_apiserver_dir}/jmeter-config/clusters.csv ${jmeter_dist_test_dir}/${perftest}

                    threads_per_pod=6  # number of threads per slave pod
                    rate_per_thread=75 # requests/s

                    # Install master and slave pods on the load driver cluster
                    if [[ ${use_distributed_load_driver} == true ]]; then
                        export KUBECONFIG="${load_conf_yaml}"
                    fi
                    ${jmeter_dist_dir}/bin/install.sh -n default -s "${apiserver_slave_pods}" -p "${keystorepwd}" -e "${ns_suffix}" ${perftest}

                    # Give everything a chance to start
                    sleep 90

                    # Run the test for an hour
                    jmeter_threads=$((apiserver_slave_pods * threads_per_pod))
                    "${jmeter_dist_dir}/bin/run.sh" -n default -t "${threads_per_pod}" -r "${rate_per_thread}" -d 3600 >"${output_file}.csv"

                    printf "\n%s - Distributed apiserver load test completed\n" "$(date +%T)"
                    test_end_time=$(date --iso-8601=seconds)
                fi

                cat "${output_file}.csv"
                grep "^TOTAL" ${output_file}.csv >results.csv

                # Send to metrics service
                sendJmeterResultsReturnError -numThreads "${jmeter_threads}" -resultsFile "results.csv" -metricsTestName "${metrics_testname}" -metrics -failOnError

                set -e
            else
                printf "\n%s - Requested number of zero worker clusters was set to %s, so no zero worker clusters will be created.\n" "$(date +%T)" "${num_zero_clusters}"
            fi
            ;;
        "registry-load")
            export KUBECONFIG="${load_conf_yaml}"
            metrics_testname="registry-load"
            output_file=registry_samples
            jmeter_dist_test_dir="${jmeter_dist_dir}/tests"

            # Clean up any old test
            jmdl_config_dir="${jmeter_dist_dir}/config"
            if [ -d "${jmdl_config_dir}" ]; then
                rm -r "${jmdl_config_dir}"
            fi
            mkdir "${jmdl_config_dir}"

            # This test does not get run as a daily run so these are just default numbers. Change as needed
            threads_per_pod=1 # number of threads per slave pod
            rate_per_thread=1 # requests/s
            registry_slave_pods=10
            duration=60

            # Install master and slave pods on the load driver cluster (the main test cluster)
            # Note armada_performance_registry_load_oauth_token has been removed from vault - we will need a new token adding to vault if this is to work
            ${jmeter_dist_dir}/bin/install.sh -s "${registry_slave_pods}" -e "${ns_suffix}" -a "-GTOKEN=${armada_performance_registry_load_oauth_token}" ${perftest}

            # Give everything a chance to start
            sleep 90

            # Run the test for duration specified
            jmeter_threads=$((registry_slave_pods * threads_per_pod))
            "${jmeter_dist_dir}/bin/run.sh" -t "${threads_per_pod}" -r "${rate_per_thread}" -d "${duration}" >"${output_file}.csv"

            printf "\n%s - Registry load test completed\n" "$(date +%T)"
            test_end_time=$(date --iso-8601=seconds)

            cat "${output_file}.csv"

            # Send to metrics service
            sendJmeterResultsReturnError -numThreads "${jmeter_threads}" -resultsFile "${output_file}.csv" -metricsTestName "${metrics_testname}" -metrics -failOnError
            ;;
        "sysbench")
            cruiserMetricsController "10s" "60s" "${perftest}"

            cd ${sysbench_dir}/bin
            ./perf-sysbench.sh -v -m -e "${ns_suffix}" -n "${perftest}" -g "${armada_registry}" -t "${test}"

            # Start cruiser metrics collection
            echo -en "${METRICS_START}" | nc localhost -w 5 "${metricsControlPort}"

            # wait for daemonset to end
            sleep 500
            # Get logs for each pod
            for p in $(kubectl get pods -n sysbench | grep sysbench- | cut -f 1 -d ' '); do
                echo ---------------------------
                echo $p
                echo ---------------------------
                kubectl logs $p -n sysbench
            done

            # This test runs in a cluster so it cannot write directly to Influxdb, so metrics are written to a file. Get the metrics files from the pods with the label specified.
            getInfluxdbMetricsFromPods name perf-sysbench-sysbench "${perftest}"

            # Stop cruiser metrics collection
            echo -en "${METRICS_STOP}" | nc localhost -w 5 "${metricsControlPort}"

            # Terminate cruiser metrics collection utility
            echo -en "${METRICS_TERMINATE}" | nc localhost -w 5 "${metricsControlPort}"

            # delete the daemonset
            kubectl delete --force daemonset "perf-${perftest}-daemonset" -n "${perftest}"
            ;;
        "incluster-apiserver")
            cruiserMetricsController "10s" "60s" "${perftest}"
            # Start cruiser metrics collection
            echo -en "${METRICS_START}" | nc localhost -w 5 "${metricsControlPort}"

            metricsTestName="${perftest}"."nodes${workers}"
            cd ${incluster_apiserver_dir}/bin

            test_start_time=$(date --iso-8601=seconds)
            replicas=20
            instances=10
            ithreads=$((replicas * instances))
            export resCSVFile="incluster-apiserver-results.csv"
            ./run_incluster_apiserver.sh -n "${perftest}" -g "${armada_registry}" -e "${ns_suffix}" -t "0" -r ${replicas} -i ${instances}
            test_end_time=$(date --iso-8601=seconds)

            # Stop cruiser metrics collection
            echo -en "${METRICS_STOP}" | nc localhost -w 5 "${metricsControlPort}"
            # Terminate cruiser metrics collection utility
            echo -en "${METRICS_TERMINATE}" | nc localhost -w 5 "${metricsControlPort}"

            # Send to metrics service
            cat ${resCSVFile}
            sendJmeterResultsReturnError -testname "${perftest}" -numThreads "${ithreads}" -resultsFile "${resCSVFile}" -metricsTestName "${metricsTestName}" -metrics -failOnError

            echo ./uninstall_incluster_apiserver.sh -n "${perftest}" -i ${instances}
            ./uninstall_incluster_apiserver.sh -n "${perftest}" -i ${instances}
            ;;
        "armada-api-load")
            cd ${armada_api_load_dir}
            echo "clusterID=${clusterID}"
            echo "cluster_type=${cluster_type}"

            # List of different requests we will run
            declare -A requests
            if [[ "${cluster_type}" == "vpc-gen2" ]]; then
                worker_request="/v2/vpc/getWorkers?cluster=${clusterID}"
            else
                worker_request="/v2/classic/getWorkers?cluster=${clusterID}"
            fi
            requests=(["get_vpcs"]="GET,/v2/vpc/getVPCs?provider=vpc-gen2" ["get_flavors"]="GET,/v1/zones?showFlavors=false&location=dal" ["get_workers"]="GET,$worker_request" ["get_albs"]="GET,/v2/alb/getClusterAlbs?cluster=${clusterID}")

            carrier="stage-us-south${host_arr[2]: -1}.containers.test.cloud.ibm.com"
            interval=300    # In seconds.  Do not set interval for longer than 1 hour or you will get RC-401,Unauthorized.
            threadLimit=200 # The throughput limit (!! requests/MINUTE !!) of each thread.
            for request_name in "${!requests[@]}"; do
                echo "$request_name - ${requests[$request_name]}"
                metricsTestName="${perftest}"."nodes${workers}.$request_name"

                # Generate requests.csv with the request
                # You can also comment out the next if block and put requests.csv of your choice in armada_api_load directory
                echo "${requests[$request_name]}" >requests.csv

                printf "\n%s - Requests in requests.csv:\n" "$(date +%T)"
                cat requests.csv
                echo

                startNumThreads=50
                stopNumThreads=150
                stepNumThreads=100
                for ((numThreads = ${startNumThreads}; numThreads <= ${stopNumThreads}; numThreads += ${stepNumThreads})); do
                    if [[ $request_name == "get_vpcs"  &&  ${numThreads} -gt 50 ]]; then
                        # Don't run this combination as we'll get rate limited https://github.ibm.com/alchemy-containers/armada-performance/issues/3200
                        continue
                    fi
                    declare -i requestPerSec=${numThreads}*${threadLimit}/60
                    printf "\n%s - Testing ${requestPerSec} req/sec\n" "$(date +%T)"
                    resCSVFile="summary_${numThreads}.out"
                    # Stop any orphaned process from previous tests
                    ./stop-armada-api-test.sh
                    # Process requests.csv
                    ./test-armada-api.sh ${carrier} ${interval} ${numThreads} ${threadLimit} ${resCSVFile}
                    summary=$(tail -1 ${resCSVFile})
                    if [[ ${summary} != *",0.00%,"* ]]; then
                        # Set return code for runAuto to fail test if errors found
                        run_auto_rc=1
                    fi
                    # Extract result and end to metrics service
                    sendJmeterResultsReturnError -testname "${perftest}" -numThreads "${numThreads}" -resultsFile "${resCSVFile}" -metricsTestName "${metricsTestName}" -metrics
                done

                # Stop any orphaned process still running
                ./stop-armada-api-test.sh

            done

            cd --
            ;;
        *)
            printf "\n%s - Unknown testcase \"${perftest}\". Ignoring\n" "$(date +%T)"
            ;;
        esac
        if [[ ${cluster_type} != "satellite-"* ]] && [ ${CARRIER_LOCK_TESTS[$perftest]+_} ]; then
            # Release the lock
            ${perf_dir}/bin/carrier-lock --kubeconfig=${perf_dir}/config/${carrierName}/admin-kubeconfig --action release
        fi

        printf "\n\n%s - ${perftest} test completed\n" "$(date +%T)"
    done
fi

# Remove sysdig if it was installed
if ${sendToSysdig}; then
    sysdig_agent_pod="sysdig-agent"
    sysdig_agent_ns="ibm-observe"
    set +e
    kubectl delete daemonset -n ${sysdig_agent_ns} ${sysdig_agent_pod} 2>/dev/null
    set -e
fi

# Add code here to get cruiser information, kubernetes logs, etc. before we delete the clusters
# <Get useful stuff before its too late>

# Delete the cluster(s)
if [[ "${delete_cluster}" == "true" ]]; then

    printf "\n%s - Deleting %s clusters and associated config files\n" "$(date +%T)" "${cluster_prefix}"
    if [[ "${num_clusters}" == "0" ]]; then
        # If num_clusters for the test is set to 0 to use existing cluster, reset to 1
        # otherwise cluster won't be deleted
        num_clusters=1
    fi
    # Might just be ZeroWorker clusters - so see if we have a real cluster
    current_clusters=$(${perf_dir}/bin/armada-perf-client2 cluster ls --json | jq --arg cnp "${cluster_prefix}1" '.[] | select(.name | startswith($cnp)) | .id' | wc -l)
    echo current_clusters: ${current_clusters}
    if ((${current_clusters} > 0)); then
        #Calculate the Kube version from the first clusters master, and export so it is tagged in metrics code - if no cluster exists then hope it was set correctly in the parameters
        gc=$(${perf_dir}/bin/armada-perf-client2 cluster get --cluster "${cluster_prefix}1" --json)
        mkv=$(echo "${gc}" | jq -r .masterKubeVersion)
        export K8S_SERVER_VERSION="${mkv}"
        ${perf_dir}/bin/armada-perf-client2 cluster rm --cluster ${cluster_prefix} --quantity ${num_clusters} --suffix --force-delete-storage --poll-interval 30s --timeout 60m --metrics
        # Add 1 to the end of cluster_prefix instead of * for the real cluster name
        rm -rf ${perf_dir}/config/${cluster_prefix}1
    fi

    # Delete any load driver clusters
    current_load_clusters=$(${perf_dir}/bin/armada-perf-client2 cluster ls --json | jq --arg cnp "${load_cluster_prefix}" '.[] | select(.name | startswith($cnp)) | .id' | wc -l)
    echo current_load_clusters: ${current_load_clusters}
    if ((${current_load_clusters} > 0)); then
        printf "\n%s - Deleting load cluster and associated config files\n" "$(date +%T)"
        ${perf_dir}/bin/armada-perf-client2 cluster rm --cluster ${load_cluster_prefix} --quantity 1 --suffix --force-delete-storage --poll-interval 30s --timeout 60m
        rm -rf ${perf_dir}/config/${load_cluster_prefix}1
    fi

    # Also delete any zero worker clusters
    current_zero_clusters=$(${perf_dir}/bin/armada-perf-client2 cluster ls --json | jq --arg cnp "${cluster_prefix_zero_workers}" '.[] | select(.name | startswith($cnp)) | .id' | wc -l)
    if ((${current_zero_clusters} > 0)); then
        printf "\n%s - Deleting %s zero worker clusters for cluster prefix: %s\n" "$(date +%T)" "${current_zero_clusters}" "${cluster_prefix_zero_workers}"
        ${perf_dir}/bin/armada-perf-client2 cluster rm --cluster ${cluster_prefix_zero_workers} --quantity ${current_zero_clusters} --suffix --force-delete-storage
        rm -rf ${perf_dir}/config/${cluster_prefix_zero_workers}*
    fi

    # For Satellite clusters, Clean up the location too
    if [[ ${cluster_type} == "satellite-"* ]]; then
        source ${armada_perf_dir}/automation/bin/cleanupSatelliteLocation.sh ${location}
    fi

    # Also delete a hollow node pod hosting cluster (i.e. if we ran tests against Kubemark cluster)
    if [[ -n "${custom_spawn_cluster}" ]]; then
        ${perf_dir}/bin/armada-perf-client2 cluster rm --cluster ${custom_spawn_cluster} --force-delete-storage
        rm -rf ${perf_dir}/config/${custom_spawn_cluster}
    fi
fi

if [[ ${run_auto_rc} == 0 ]]; then
    printf "\n%s - Test(s) Completed\n" "$(date +%T)"
else
    printf "\n%s - Test(s) Failed with return code ${run_auto_rc}\n" "$(date +%T)"
fi

exit $((${run_auto_rc}))
