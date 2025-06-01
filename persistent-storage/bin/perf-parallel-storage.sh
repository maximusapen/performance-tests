#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Test client directories
perf_dir=/performance
influxdb_metrics_dir=${perf_dir}/metrics

#Functions

getMetricsFromJob() {
    # Label to identify the pods which hold the influxdb results files (the data is expected to be in the /performance/metrics pod dir in a file of the form sysbench0.json)
    jobName=$1
    nameSpace=$2
    # Time to wait for the metrics file to be generated before giving up. The function returns when the metrics file appears or the job itself is completes.
    waitMin=$3
    # The number of this particular job within its parallel group
    jobNumber=$4
    # The blockSize tested by the job
    jobBlockSize=$5
    # The read/write mode tested by the job
    jobRWMode=$6

    set +e
    jobPodName=$(kubectl get pod -n ${nameSpace} --no-headers -l job-name=${jobName} | awk '{print $1}')
    jobNode=$(kubectl describe pod -n ${nameSpace} ${jobPodName} | grep Node: | awk '{print $2;}')
    printf "Waiting up to ${waitMin} minutes for the metrics file to be written .. or the job to complete. \nTracking pod: ${jobPodName} in namespace: ${nameSpace} on node: ${jobNode}\n"

    # Create the file name out of all the specific settings for this particular job
    metricsFileName=${influxdb_metrics_dir}/${nameSpace}-${jobRWMode}-${jobBlockSize}-${jobNumber}.json

    waitCount=1
    until [[ $waitCount -gt ${waitMin} ]]; do
        #Check the Pod has actually started first
        status=$(kubectl get pod -n ${nameSpace} ${jobPodName} -o jsonpath='{.status.phase}')
        if [ "${status}" == "Pending" ] || [ "${status}" == "ContainerCreating" ]; then
            printf "Pod: ${jobPodName} has not started yet, will continue waiting. Current state: ${status}\n"
        else
            printf "Pod: ${jobPodName} is now in state: ${status}\n"
            # Stop waiting when the results file becomes available
            jobOutputFile="${nameSpace}1.json"
            metricsFileExists=$(kubectl exec -n ${nameSpace} ${jobPodName} -- sh -c "if [ -e "/performance/metrics/${jobOutputFile}" ] ; then echo \"true\"; else echo \"false\"; fi ")
            if [ "${metricsFileExists}" == "true" ]; then
                printf "Metrics file found in pod: ${jobPodName}\n"
                sleep 10 # to ensure the file is fully written
                printf "Copying metrics file to ${influxdb_metrics_dir}\n"
                kubectl cp ${nameSpace}/${jobPodName}:/performance/metrics/${jobOutputFile} ${metricsFileName}
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
        ((waitCount++))
        sleep 60
    done
    if [ -f "${metricsFileName}" ]; then
        printf "\n - Metrics file successfully copied to perf client: ${metricsFileName}"
    else
        printf "\n - No metrics file %s not found - so metrics will not be written to Influxdb.\n" "${metricsFileName}"
    fi
    set -e
}

combineAndUploadToInfluxDB() {

    # The file prefix of the collection of metrics files we
    # want to combine and upload to Influx.
    prefix=$1

    # Use the parallel specific send command to agregate all of the 
    # parallel runs into a single set of metrics to be uploaded.
    ${perf_dir}/bin/send-parallel-files-to-Influx -metricsdir ${influxdb_metrics_dir} -testname ${prefix} -verbose
}

#Defaults
registry=""
environment=""
namespace="default"
storagemode="ReadWriteMany"
storagesize="20Gi"
storageclass="ibmc-block-bronze"
scheduler="default-scheduler"
metrics=false
verbose=false
podcount="1"
ioPing=true

jobNamePrefix=parallel-storage-

while test $# -gt 0; do
    case "$1" in
    -h | --help)
        echo "perf-parallel-storage - runs persistent storage performance tests in parallel"
        echo " "
        echo "perf-parallel-storage [options]"
        echo " "
        echo "options:"
        echo "-h, --help                  	show brief help"
        echo "-t, --testname testname   	Name of test in Jenkins - only used if sending alerts to RazeeDash"
        echo "-e, --environment environment     Registry environment namespace (e.g stage1, stage2, etc.)"
        echo "-g, --registry registry_url	image registry location"
        echo "-n, --namespace k8s_namespace	kubernetes namespace for deployment"
        echo "-m, --metrics			request results are sent to IBM Cloud monitoring service"
        echo "-v, --verbose			log test results to stdout"
        echo "-s, --size size			persistent storage size (e.g. 20Gi)"
        echo "-p, --podcount podcount		Number of concurrent test pods"
        echo "-c, --class storage_class	persistent storage class name (e.g. ibmc-block-bronze)"
        echo "-x, --scheduler scheduler_name    pod scheduler"
        echo "-al, --affinity-label label The affinity label to match against for pod affinity checks"
        echo "-av, --affinity-values value_list The comma separated list of values for the supplied affinity label"
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
    -p | --podcount)
        shift
        if test $# -gt 0; then
            podcount=$1
        else
            echo "Pod count not specified, using default of $podcount"
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
    -al | --affinity-label)
        shift
        if test $# -gt 0; then
            affinityLabel=$1
        else
            echo "affinity label not specified"
            exit 1
        fi
        shift
        ;;
    -av | --affinity-values)
        shift
        if test $# -gt 0; then
            affinityValuesParam=$1
            IFS=',' read -r -a affinityValues <<< "$affinityValuesParam"
        else
            echo "affinity value list not specified"
            exit 1
        fi
        shift
        ;;
    *)
        break
        ;;
    esac
done

if [[ -z $affinityLabel ]] && [[ -n $affinityValuesParam ]]; then
    echo "Affinity values specified without an affinity label"
    exit 1
fi

if [[ -n $affinityLabel ]] && [[ -z $affinityValuesParam ]]; then
    echo "Affinity label specified without any affinity values"
    exit 1
fi

if [[ $storageclass == *"block"* ]] || [[ $storageclass == *"rbd"* ]]; then
    storagemode="ReadWriteOnce"
fi

echo Namespace: ${namespace}
echo Verbose: ${verbose}
echo Pod scheduler: ${scheduler}
echo Storage Class Name: ${storageclass}
echo Storage Size: ${storagesize}
echo Test Pod Count: ${podcount}
echo Metrics: ${metrics}

if [[ -n ${affinityValuesParam} ]]; then
    echo Pod Affinity Label: ${affinityLabel}
    echo Pod Affinity Values: ${affinityValuesParam}
fi

if [[ -n ${registry} ]]; then
    echo Registry: ${registry}
    setRegistry="--set image.registry=${registry}"
fi

if [[ -n ${environment} ]]; then
    echo Registry Environment: ${environment}
    setEnvironment="--set image.name=armada_performance_${environment}/persistent-storage"
fi

# Clear any old metrics files out of Influx directory. Instead of doing an rm here
# do a move instead to avoid any possible nasty side effects from a mis-called rm.
mkdir -p ${influxdb_metrics_dir}/old
mv -f ${influxdb_metrics_dir}/${namespace}*.json ${influxdb_metrics_dir}/old

# In the podcount = 1 case run the test with both fio and ioping
# enabled. But avoid running ioping in multi-pod runs just to cut
# down on the overall run time for our complete test scenario.
if [ $podcount -gt 1 ]; then
    # Skip ioping test
    ioPing=false
fi

# Create a new config map so that we can pass in the supplied parameters to the job(pod) creation
perf_ps_dir="/var/perfps/${storageclass}-${storagesize}"
kubectl delete configmap perf-persistent-storage-config --ignore-not-found=true --namespace=${namespace}
kubectl create configmap perf-persistent-storage-config --namespace=${namespace} --from-literal=PERF_PS_VERBOSE=${verbose} --from-literal=PERF_PS_METRICS=${metrics} --from-literal=PERF_PS_TESTNAME=${testname} --from-literal=PERF_PS_DIR="${perf_ps_dir}" --from-literal=PERF_PS_PODCOUNT="${podcount}" --from-literal=PERF_PS_IOPING=${ioPing}

# Prefix to be appended with current test counter to avoid pvc
# name clashes on parallel test runs.
pvcNamePrefix=perf-pvc-

# Send start metrics gathering control byte
metricsControlPort=10569
echo -en "\x1" | nc localhost -w 5 "${metricsControlPort}"

# Loop through all of the read/write modes we want
# to include in the test scenario.
declare -a rwModes=("randread" "write")
for rwMode in "${rwModes[@]}"; do

    # Loop through all of the block sizes we want
    # to include in the test scenario.
    declare -a blockSizes=("4k" "16k" "64K")
    for bs in "${blockSizes[@]}"; do

        # Delete any existing jobs that have the same job prefix. This
        # should catch all previous instances of the same job.
        echo "Attempting to cleanup any previous runs."
        helm list --namespace=${namespace} --short | awk -v pattern="${jobNamePrefix}" '$1 ~ pattern' | awk '{print $1}' | xargs -L1 helm uninstall --namespace=${namespace}

        # The pvcs can take a while to actually delete - so we need to check they 
        # have gone before we try install again, otherwise the install will fail.
        counter=0
        attempts=60
        while [ $counter -lt $attempts ]; do
            # Check to see if any resources remain for the whole of the pod range that we intend to start.
            # Even if the run before this was more or less pods we should check that any that might clash
            # with the pods we need for this run are gone.
            remaining_resources="$(kubectl get pvc --namespace ${namespace} | awk -v pattern="${pvcNamePrefix}" '$1 ~ pattern' | awk '{print $1}')"
            if [ -n "${remaining_resources}" ]; then
                echo "Resources still exist. ${counter}/${attempts} tests completed; retrying."
                echo "${remaining_resources}" 1>&2
                ((++counter))
                sleep 10
            else
                break
            fi
        done

        # We hit our attempt count without deleting all of the pvcs so exit with
        # output from `describe pvc` to hopefully help with debug.
        if [ $counter -eq $attempts ]; then
            echo "Persistent Volume Claims failed to delete in time. Exiting";

            # NOTE: kubectl will treat `perf-pvc-` here as a pattern and will match
            #       it against all pvcs with that prefix.
            kubectl describe pvc --namespace=${namespace} ${pvcNamePrefix} 2>/dev/null
            
            exit 1
        fi

        # Default to a generic "match every node" setup for our pod affinity    
        # in case no custom label and values were supplied. This should mean
        # The afffinity logic doesn't get in the way of any other scheduling
        # logic i.e. using a different scheduler
        setAffinityLabel="--set pod.affinity.label=ibm-cloud.kubernetes.io/region" # TODO: Is this a reasonable default?
        setAffinityValue="--set pod.affinity.value=us-south"
            
        # Start our, possibly multiple, test runs. The helm install will create a separate pod
        # and pvc for each iteration so each test should be running against its own storage.
        echo "Starting ${podcount} test runs..."
        podCounter=1
        affinityValueIndex=0
        while [ $podCounter -le $podcount ]; do

            if [[ -n ${affinityValuesParam} ]]; then
                # Round robin around our available values when starting each run to try
                # and get a good distribution across our cluster.
                affinityValue=${affinityValues[$affinityValueIndex]}
                setAffinityValue="--set pod.affinity.value=${affinityValue}"
                ((++affinityValueIndex))
                if [ $affinityValueIndex -eq ${#affinityValues[@]} ]; then
                    affinityValueIndex=0
                fi

                # As we are using a custom value make sure to also use
                # the custom label. Enforcement during parameter parsing
                # means we should never get one without the other.
                setAffinityLabel="--set pod.affinity.label=${affinityLabel}"
            fi

            # Use helm to install the chart and execute the tests as a kubernetes job on the cluster
            # NOTE: We need to use the iteration counter in the jobName and pvcName to avoid clashes
            #       between parallel runs.
            helm install ${jobNamePrefix}${podCounter}of${podcount} ../imageDeploy/parallel-storage --namespace=${namespace} --set blockSize=${bs} --set rwMode=${rwMode} --set metricsPrefix=${METRICS_PREFIX} --set k8sVersion=${K8S_SERVER_VERSION} --set pvc.accessMode=${storagemode} --set pvc.storageSize=${storagesize} --set pvc.storageClassName=${storageclass} --set pvc.name=${pvcNamePrefix}${podCounter} --set pod.scheduler=${scheduler} ${setAffinityLabel} ${setAffinityValue} ${setRegistry} ${setEnvironment}
            ((++podCounter))
        done

        # Output our pod details after a short wait to let them all get 
        # created. Use `-o wide` so that we get node details for each pod.
        sleep 60
        echo "Test pods for parallel runs:"
        kubectl -n ${namespace} get pods -o wide

        echo "Waiting for ${podcount} test runs to complete..."
        podCounter=1
        waitTime=120
        while [ $podCounter -le $podcount ]; do
            # Wait for metrics file to be ready
            getMetricsFromJob ${jobNamePrefix}${podCounter}of${podcount}-job ${namespace} ${waitTime} ${podCounter} ${bs} ${rwMode}

            # Output job logs in case we need them for debug
            printf "\n ----------- Historic logs from job: ${jobNamePrefix}${podCounter}of${podcount}-job ----------- \n"
            kubectl logs job/${jobNamePrefix}${podCounter}of${podcount}-job -n "${namespace}"
            printf "\n ---------------------------------------------------------------------- \n\n"

            # After waiting for the first parallel job reduce our wait time as 
            # hopefully all of the other jobs should be finishing around the same
            # time. Otherwise our wait here would get ridiculously long.
            waitTime=5

            ((++podCounter))
        done

        # Once we have completed waiting for all of our test group to finish we 
        # can now collect their results together for uploading to Influx DB.
        echo "Uploading test run files to Influx..."
        combineAndUploadToInfluxDB ${nameSpace}-${rwMode}-${bs}-

    done # block size

done # rw mode

# Send stop metrics gathering control byte
echo -en "\x2" | nc localhost -w 5 "${metricsControlPort}"
