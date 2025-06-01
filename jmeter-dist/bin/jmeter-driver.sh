#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019, 2023 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
die() {
    printf '%s\n' "$1" >&2
    exit 1
}

determineSlavePodCPURequest() {
    # Get the total allocatable cpu in the cluster for nodes that are to host slave pods
    allocatable_cpu=$(kubectl get nodes -l use=slaves -o jsonpath='{.items[*].status.allocatable.cpu}')
    IFS=' ' read -r -a nodecpu_array <<<${allocatable_cpu}
    nodeCount="${#nodecpu_array[@]}"
    for c in "${nodecpu_array[@]}"; do
        unit="${c: -1}"
        val="${c::-1}"
        case "${unit}" in
        "m")
            total_allocatable_cpu=$((total_allocatable_cpu + val))
            ;;
        *)
            total_allocatable_cpu=$((total_allocatable_cpu + (val * 1000)))
            ;;
        esac
    done

    if [[ -z ${slavePods} ]]; then
        # Calculate number of pods using fixed default of 1000m cpu request
        slavePods=$((total_allocatable_cpu / nodeCount / 1000 * nodeCount))
        slave_pod_cpu_request="1000m"
    else
        # Try our best to balance the requested number of pods across our nodes.
        # This is best effort and works best with larger number of pods.
        # Smaller numbers can hit issues with unschedulable pods.
        while
            terminating_pods=$(kubectl get pods -n ${namespace} -o wide | grep Terminating)
            [[ -n ${terminating_pods} ]]
        do
            printf "%s - Waiting for pods to terminate\n" "$(date +%T)"
            sleep 15
        done

        # Get the current total number of cpu requests in the cluster
        pod_cpu_requests=$(kubectl get pods --all-namespaces -o=jsonpath="{range .items[*]}{range .spec.containers[*]}{.resources.requests.cpu}{','}{end}")

        IFS=', ' read -r -a cpu_requests_array <<<${pod_cpu_requests}
        for cpur in "${cpu_requests_array[@]}"; do
            if [[ -n ${cpur} ]]; then
                unit="${cpur: -1}"
                val="${cpur::-1}"
                case "${unit}" in
                "m")
                    millis=$((millis + val))
                    ;;
                *)
                    millis=$((millis + (val * 1000)))
                    ;;
                esac
            fi
        done

        # Determine how much is remaining for our pods - leave some contingency (10%) for existing unbalanced scheduling
        remaining_allocatable_cpu=$((total_allocatable_cpu - millis))

        # Esnure it will fit on a node
        if ((slavePods < nodeCount)); then
            perPod=${nodeCount}
        else
            perPod=${slavePods}
        fi

        alloc_per_pod=$((remaining_allocatable_cpu / perPod))
        slave_pod_cpu_request="$((alloc_per_pod * 80 / 100))m"
    fi
}
jmeter_dist_dir="/performance/armada-perf/jmeter-dist"
config_dir="${jmeter_dist_dir}/config"

namespace="default"
registry=""
environment=""
mode="slave"
configMapName="jmeter-dist-config"

slavePods=""
jksPassword=""

jmeterargs="-GRAMPUPSECS=0 "
# Alternative configurations
#jmeterargs="$jmeterargs -Jmode=StrippedDiskStore "
#jmeterargs="$jmeterargs -Jmode=Statistical -Dnum_sample_threshold=250 "
jvmargs="-Xmx2048M -Xms2048M"
#jvmargs="-Xmx8192M -Xms8192M -verbose:gc"

while test $# -gt 0; do
    case "$1" in
    -h | --help)
        echo "jmeter-driver - runs Kubernetes based distributed JMeter load testing"
        echo " "
        echo "jmeter-driver [options] args"
        echo " "
        echo "options:"
        echo "-h, --help                        show brief help"
        echo "-c, --config <folder>             folder containing runtime configuration files"
        echo "-t, --threads <N>                 number of JMeter slave threads to be run on each pod"
        echo "-r, --rate <req/s>                target throughput of each thread"
        echo "-d, --duration <N>                test duration in seconds"
        echo "-e, --environment <environment>   registry environment namespace (e.g stage1, stage2, etc.)"
        echo "-g, --registry <registry_url>     image registry location"
        echo "-m, --mode [slave|standalone]     Defines the pod deployment mode (slave is the default)."
        echo "-n, --namespace <k8s_namespace>   Kubernetes namespace for deployment"
        echo "-p, --password <keystore_pwd>     Password for Java keystore holding authentication credentials"
        echo "-s, --slaves <N>                  specify number of JMeter slave replicas(pods) which are to be used as JMeter load generators"
        echo "-j, --jvmargs <args>              specify the jmeter JVM args. Default: ${jvmargs}"
        exit 0
        ;;
    -c | --config)
        shift
        if test $# -gt 0; then
            config_dir=$1
        else
            echo "Runtime configuration folder not specified"
            exit 1
        fi
        shift
        ;;
    -t | --threads)
        shift
        if test $# -gt 0; then
            threads=$1
            jmeterargs="${jmeterargs}-GTHREAD=${threads} "
        else
            echo "Number of JMeter threads not specified"
            exit 1
        fi
        shift
        ;;
    -r | --rate)
        shift
        if test $# -gt 0; then
            rate=$(($1 * 60)) # JMeter throughput is per min
            jmeterargs="${jmeterargs}-GTPUTMINS=${rate} "
        else
            echo "Target rate/throughput not specified"
            exit 1
        fi
        shift
        ;;
    -d | --duration)
        shift
        if test $# -gt 0; then
            duration=$1
            jmeterargs="${jmeterargs}-GRUNLENSEC=${duration} "
        else
            echo "Test duration not specified"
            exit 1
        fi
        shift
        ;;
    -s | --slaves)
        shift
        if test $# -gt 0; then
            slavePods=$1
        else
            echo "Number of slave replicas not specified"
            exit 1
        fi
        shift
        ;;
    -m | --mode)
        shift
        if test $# -gt 0; then
            mode="$1"
            if [[ $mode != "slave" && $mode != "standalone" ]]; then
                echo "Invalid mode specified: ${mode}"
                exit 1
            fi
            if [[ $mode == "standalone" ]]; then
                configMapName="jmeter-${1}-config"
            fi

        else
            echo "Mode not specified"
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
    -p | --password)
        shift
        if test $# -gt 0; then
            jksPassword=$1
        else
            echo "Keystore password not specified"
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
    -j | --jvmargs)
        shift
        if test $# -gt 0; then
            jvmargs=$1
        else
            echo "JVM args not specified"
            exit 1
        fi
        shift
        ;;
    *)
        jmeterargs="${jmeterargs} "$*
        break
        ;;
    esac
done

echo Namespace: ${namespace}
echo JMeter Args: $(echo ${jmeterargs} | sed 's/-GTOKEN=.[[:print:]]*/ /')

if [[ -n ${registry} ]]; then
    echo Registry: ${registry}
    setRegistry="--set image.registry=${registry}"
fi

if [[ -n ${environment} ]]; then
    echo Registry Environment: ${environment}
    setEnvironment="--set image.name=armada_performance_${environment}/jmeter-dist"
fi

determineSlavePodCPURequest

if [[ -n ${slavePods} ]]; then
    echo Slave Replicas: ${slavePods}
    setSlaveReplicas="--set slaveReplicas=${slavePods}"
fi

setSlavePodCPURequest="--set jmeter.slave.cpurequest=${slave_pod_cpu_request}"

# Delete any existing job from previous runs
helm uninstall jmeter-dist --namespace ${namespace} 2>/dev/null
helm uninstall jmeter-standalone --namespace=${namespace} 2>/dev/null

# Create a new config map so that we can pass in the supplied parameters to the pod creation
kubectl delete configmap "${configMapName}" --ignore-not-found=true --namespace=${namespace}

while (true); do
    count=$(kubectl get pods --namespace=${namespace} | egrep "jmeter-(dist|standalone)" | wc -l | awk '{print $1}')
    if [[ count -eq 0 ]]; then
        break
    fi
    echo "$count pods still terminating"
    sleep 10
done

kubectl create configmap "${configMapName}" --namespace=${namespace} \
    --from-literal=JMETER_ARGS="${jmeterargs}" \
    --from-literal=SLAVE_PODS="${slavePods}" \
    --from-literal=KEYSTORE_PWD="${jksPassword}" \
    --from-literal=JVM_ARGS="${jvmargs}" \
    --from-file ${config_dir}

if [[ $? -ne 0 ]]; then
    echo "Failed to create configmap"
    exit 1
fi

# Use helm to install the chart and execute the tests as a kubernetes job on the cluster
# Sometimes need to retry the helm install if it was issued just as the openvpn restart was happening in the new load cluster

exitcode=1
loops=0
maxLoops=5

# Set if running in slave mode or standalone
if [[ $mode == "slave" ]]; then
   jmeter_type=jmeter-dist
else
   jmeter_type=jmeter-standalone
fi      

while [[ $exitcode -ne 0 && $loops < $maxLoops ]]; do
    echo "helm install try loop: $loops"
    helm install ${jmeter_type} "${jmeter_dist_dir}"/imageDeploy/${jmeter_type} --namespace=${namespace} --wait --timeout=300s ${setSlaveReplicas} ${setSlavePodCPURequest} ${setRegistry} ${setEnvironment}
    exitcode=$?
    ((loops++))
    if [[ $exitcode -ne 0 ]]; then
        echo "Helm install failed, printing pod status:"
        kubectl get pods -A -o=wide
        kubectl describe pod --namespace "${namespace}"
        if [[ $loops == $maxLoops ]]; then
            # Failed install but do not uninstall for investigation
            echo "Helm install failed too many times. Exiting with Error"
            exit $exitcode
        else
            # Uninstall before retry
            echo "Helm install exit code $?.  Uninstall before retry."
            helm uninstall ${jmeter_type} --namespace ${namespace} 2>/dev/null
            # Sleep for a bit before retry to make sure all resources are removed
            sleep 60
        fi
    fi
done