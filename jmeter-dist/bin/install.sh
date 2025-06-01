#!/bin/bash

# Example of installing a test on a cluster.

die() {
    printf '%s\n' "$1" >&2
    exit 1
}

namespace=""
threads=""
rate=""
duration=""
environment=""
slavePods=""
jksPassword=""
mode="-m slave"
jvmargs=""

jmeter_dist_dir="/performance/armada-perf/jmeter-dist"

# KUBECONFIG needs to be set before we attempt the installation
if [[ -z ${KUBECONFIG} ]]; then
    die 'ERROR: KUBECONFIG must be set.'
fi

while test $# -gt 0; do
    case "$1" in
    -h | --help)
        echo "install - installs Kubernetes based distributed JMeter load testing"
        echo " "
        echo "install [options] testname"
        echo " "
        echo "options:"
        echo "-h, --help                        show brief help"
        echo "-t, --threads <N>                 number of JMeter slave threads to be run on each pod"
        echo "-r, --rate <N>                    target throughput of each thread in requests/s"
        echo "-d, --duration <N>                test duration in seconds"
        echo "-e, --environment <environment>   registry environment namespace (e.g stage1, stage2, etc.)"
        echo "-m, --mode [slave|standalone]     Defines the pod deployment mode (slave is the default)."
        echo "-n, --namespace <k8s_namespace>   Kubernetes namespace for deployment"
        echo "-s, --slaves <N>                  specify number of JMeter slave replicas(pods) which are to be used as JMeter load generators"
        echo "-p, --password <keystore_pwd>     Password for Java keystore holding authentication credentials"
        echo "-j, --jvmargs <args>              specify the jmeter JVM args. Default: -Xmx1024M -Xms1024M"
        exit 0
        ;;
    -t | --threads)
        shift
        if test $# -gt 0; then
            threads="-t $1"
        else
            echo "Number of JMeter threads not specified"
            exit 1
        fi
        shift
        ;;
    -r | --rate)
        shift
        if test $# -gt 0; then
            rate="-r $1"
        else
            echo "Target rate/throughput not specified"
            exit 1
        fi
        shift
        ;;
    -d | --duration)
        shift
        if test $# -gt 0; then
            duration="-d $1"
        else
            echo "Test duration not specified"
            exit 1
        fi
        shift
        ;;
    -e | --environment)
        shift
        if test $# -gt 0; then
            environment="-e $1"
        else
            echo "registry namespace environment not specified"
            exit 1
        fi
        shift
        ;;
    -s | --slaves)
        shift
        if test $# -gt 0; then
            slavePods="-s $1"
        else
            echo "Number of slave replicas not specified"
            exit 1
        fi
        shift
        ;;
    -n | --namespace)
        shift
        if test $# -gt 0; then
            namespace="-n $1"
        else
            echo "Namespace not specified"
            exit 1
        fi
        shift
        ;;
    -m | --mode)
        shift
        if test $# -gt 0; then
            mode="-m $1"
            if [[ $mode != "-m slave" && $mode != "-m standalone" ]]; then
                echo "Invalide mode specified: ${mode}"
                exit 1
            fi
        else
            echo "Mode not specified"
            exit 1
        fi
        shift
        ;;
    -p | --password)
        shift
        if test $# -gt 0; then
            jksPassword="$1"
            jksPasswordArg="-p ${jksPassword}"
        else
            echo "Keystore password not specified"
            exit 1
        fi
        shift
        ;;
    -j | --jvmargs)
        shift
        if test $# -gt 0; then
            jvmargs="-jvmargs $1"
        else
            echo "JVM args not specified"
            exit 1
        fi
        shift
        ;;
    -a | --jmeterargs)
        shift
        if test $# -gt 0; then
            jmeterargs="$1"
        else
            echo "JMeter args not specified"
            exit 1
        fi
        shift
        ;;
    *)
        test_name=$*
        break
        ;;
    esac
done

if [[ -z "${test_name}" ]]; then
    die 'ERROR: test name must be specified.'
fi

# Clean up any old test
config_dir="${jmeter_dist_dir}/config"
if [ -d "${config_dir}" ]; then
    sudo rm -r "${config_dir}"
fi
mkdir "${config_dir}"

# Copy test specific configuration files to the config directory
test_dir="${jmeter_dist_dir}/tests/${test_name}"
cp "${test_dir}"/* "${config_dir}"

if [[ -n "${jksPassword}" ]]; then
    keytool -list -keystore "${config_dir}/cert.jks" -storepass "${jksPassword}" >/dev/null
    if [[ $? != 0 ]]; then
        die "ERROR: Cannot access Java keystore - check password"
    fi
else
    # Disable the keystore configuration if needed
    sed -i -r 's/(testname="Keystore Configuration" enabled=)"true"/\1"false"/' "${config_dir}/test.jmx"
fi

# Taint and label nodes such that 1 node hosts the jmeter-dist master pod, and the rest slaves
allNodes=$(kubectl get nodes --no-headers | awk '{print $1}')
master_done=""
for node in ${allNodes}; do
    if [[ -z ${master_done} ]]; then
        kubectl taint nodes "${node}" key=master:NoSchedule --overwrite=true
        kubectl label node "${node}" use=master --overwrite=true
        master_done="true"
    else
        kubectl label node "${node}" use=slaves --overwrite=true
    fi
done
kubectl get nodes -L use

# Install on cluster using helm
"${jmeter_dist_dir}/bin/jmeter-driver.sh" ${namespace} ${slavePods} ${duration} ${threads} ${rate} ${jksPasswordArg} ${environment} ${mode} ${jvmargs} ${jmeterargs}
