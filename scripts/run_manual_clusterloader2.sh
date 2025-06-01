#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# This script take the cluster name as an input parameter. It should be run on the performance client which created the cluster (so the cluster config files are present).
# It is run using : nohup ./run_manual_clusterloader2.sh <clustername> &

helpInfo() {
    echo
    echo "${0##*/} runs a selection of k8s end to end clusterloader2 tests on a perf client. It avoids the need to use Jenkins which is unreliable for long running, large clusters tests."
    echo "The script assumes the namespace and registry secrets have already been set up as done in a Jenkins run."
    echo "The script must be run on the performance client that created the cluster so the cluster configuration files are present."
    echo
    echo "Output will be written to nohup1.txt. The script keeps the last two runs - renaming them at the start of the run to nohup2.txt and nohup3.txt"
    echo "The script will run the kubernetes performance load and density tests in the background .. so you can safely log off the machine and the tests will continue to run."
    echo "Check the results file for successful/failed runs using: cat nohup1.txt | grep -A 5 'Success\|Failed'"
    echo "The script does not collect metrics data. Carrier data can be obtained later from Grafana."
    echo " "
}

usage() {
    echo
    echo "Usage: nohup ${0##*/} -c <cluster_name> <options>&"
    echo "e.g. nohup ${0##*/} -c myCluster1 -l -d &"
    echo " "
    echo "options:"
    echo "-c, --cluster     Cluster_name                          Name of the cluster to use for the tests"
    echo "-l, --load        Run standard load tests:          [sig-scalability] Load capacity [Feature:Performance]       30 pods per node { ReplicationController} with  secrets and configmaps"
    echo "-d, --density     Run standard density tests:       [sig-scalability] Density       [Feature:Performance]       30 pods per node using { ReplicationController} with secrets and configmaps"
    echo
    exit

}

stdLoadTests=false
stdDenityTests=false
testRun=false

if [ -z ${1} ]; then
    helpInfo
    usage
fi

while test $# -gt 0; do
    case "$1" in
    -c | --cluster)
        shift
        if test $# -gt 0; then
            cluster_name=$1
        else
            printf '\nError : Cluster name must be specified!\n\n'
            usage
            exit 1
        fi
        shift
        ;;
    -l | --load)
        stdLoadTests=true
        shift
        ;;
    -d | --density)
        stdDenityTests=true
        shift
        ;;
    -?*)
        printf 'WARN: Unknown option (ignored): %s\n' "$1" >&2
        shift
        ;;
    *)
        break
        ;;
    esac
done

if [ -z ${cluster_name+x} ]; then
    printf '\nError : Cluster name must be specified!\n\n'
    usage
    exit 1
fi

if [ "${testRun}" = false ]; then

    #backup old runs
    echo
    echo backing up old runs
    rm nohup3.txt
    mv nohup2.txt nohup3.txt
    mv nohup1.txt nohup2.txt
    mv nohup.out nohup1.txt

    echo
    echo Running tests on cluster: ${cluster_name}
    echo Results will be written to nohup1.txt
    echo To see final test results run: cat nohup1.txt | grep 'SUCCESS!\|FAIL!'
else
    echo
    echo Running dry run of tests on cluster: ${cluster_name}
fi

perf_dir=/performance
armada_perf_dir=${perf_dir}/armada-perf
k8s_e2e_perf_dir=${armada_perf_dir}/k8s-e2e-perf-large

export GOPATH=${perf_dir}
export KUBE_ROOT="${GOPATH}"/src/k8s.io/perf-tests/clusterloader2
export KUBECONFIG="/performance/config/${cluster_name}/kube-config-dal09-${cluster_name}.yml"
export KUBE_MASTER_IP=$(sed -n '/server:/p' "${KUBECONFIG}" | rev | cut -d/ -f1 | rev | head -1)
export KUBE_MASTER=$(echo ${KUBE_MASTER_IP} | cut -d: -f1)

cluster_config_dir="${perf_dir}"/config/${cluster_name}
export LOG_DUMP_SSH_KEY="${cluster_config_dir}"/.ssh/id_rsa
export LOG_DUMP_SSH_USER=root

cd "${KUBE_ROOT}"

printf "%s - Building Clusterloader....\n" "$(date +%T)"
go build -o clusterloader './cmd/'
printf "%s - Clusterloader build completed.\n" "$(date +%T)"

k8s_report_dir="${k8s_e2e_perf_dir}"/reports
k8s_log_dir="${k8s_e2e_perf_dir}"/logs

# We need to create the namespace before the test so that we can create the secret in it
kubectl create namespace monitoring

e2eTestRun() {
    #function argument: K8s end to end test pattern
    test_config_dir=$1

    echo
    echo Starting K8s performance test: ${_e2e_test_Pattern}
    echo

    # Use the Kubernetes master API server local proxy
    master_internal_ip=172.21.0.1
    test_start_time=$(date --iso-8601=seconds)
    # If you want to scrape the etcd data from Prometheus in the cluster as well as the apiserver data you need to remove --master_internal_ip in the following command and follow the details in this Box note https://ibm.ent.box.com/notes/649560103756
    "${KUBE_ROOT}"/clusterloader --v=2 --kubeconfig "${KUBECONFIG}" --tear-down-prometheus-server=false --alsologtostderr --provider skeleton --report-dir "${k8s_report_dir}" --log_dir "${k8s_log_dir}" --testconfig "${test_config_dir}/config.yaml" --testoverrides "${test_config_dir}/overrides_large.yaml" --enable-prometheus-server --prometheus-scrape-kube-proxy=false --prometheus-scrape-kubelets=false --prometheus-scrape-node-exporter=false --master-internal-ip ${master_internal_ip}

    test_end_time=$(date --iso-8601=seconds)

    sudo chown -R jenkins:"Domain Users" "${k8s_report_dir}"

}

if [ "${stdLoadTests}" = true ]; then
    echo
    echo Starting k8s Load tests
    e2eTestRun "${KUBE_ROOT}/testing/load"
fi

if [ "${stdDenityTests}" = true ]; then
    echo
    echo Starting k8s Density tests
    e2eTestRun "${KUBE_ROOT}/testing/density"
fi

echo
echo Finished
