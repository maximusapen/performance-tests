#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# This script take the cluster name as an input parameter. It should be run on the performance client which created the cluster (so the cluster config files are present).
# It is run using : nohup ./run_manual_e2e.sh <clustername> &

helpInfo() {
    echo
    echo "${0##*/} runs a selection of k8s end to end tests on a perf client. It avoids the need to use Jenkins which is unreliable for long running, large clusters tests."
    echo "The script assumes the namespace and registry secrets have already been set up (i.e. the K8s_e2e tests have been run at least once from Jenkins)."
    echo "The script must be run on the performance client that created the cluster so the cluster configuration files are present."
    echo
    echo "Output will be written to nohup1.txt. The script keeps the last two runs - renaming them at the start of the run to nohup2.txt and nohup3.txt"
    echo "The script will run the kubernetes performance load and density tests in the background .. so you can safely log off the machine and the tests will continue to run."
    echo "Check the results file for successful/failed runs using: cat nohup1.txt | grep -A 5 'SUCCESS!\|FAIL!'"
    echo "The script does not collect metrics data. Carrier data can be obtained later from Grafana."
    echo " "
}

usage() {
    echo
    echo "Usage: nohup ${0##*/} -c <cluster_name> <options>&"
    echo "e.g. nohup ${0##*/} -c myCluster1 -l -d &"
    echo " "
    echo "options:"
    echo "-c, --cluster     Cluster_name	                  Name of the cluster to use for the tests"
    echo "-l, --load        Run standard load tests:          [sig-scalability] Load capacity [Feature:Performance]       30 pods per node { ReplicationController} with 0 secrets, 0 configmaps and 0 daemons"
    echo "-d, --density     Run standard density tests:       [sig-scalability] Density       [Feature:Performance]       30 pods per node using { ReplicationController} with 0 secrets, 0 configmaps and 0 daemons"
    echo "-s, --secrets     Run standard secret tests:        [sig-scalability] Load capacity [Feature:ManualPerformance] 30 pods per node {extensions Deployment} with 2 secrets, 0 configmaps and 0 daemons"
    echo "-m, --maps        Run standard configmap tests:     [sig-scalability] Load capacity [Feature:ManualPerformance] 30 pods per node {extensions Deployment} with 0 secrets, 2 configmaps and 0 daemons"
    echo "-a, --all:        Run all of the tests above"
    echo "-t, --test:       Dry run to check which tests would be executed - nothing is actually run"
    echo
    exit

}

stdLoadTests=false
stdDenityTests=false
stdConfigmapTests=false
stdSecretTests=false
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
    -d | --denisty)
        stdDenityTests=true
        shift
        ;;
    -m | --configmap)
        stdConfigmapTests=true
        shift
        ;;
    -s | --secrets)
        stdSecretTests=true
        shift
        ;;
    -a | --all)
        stdLoadTests=true
        stdDenityTests=true
        stdConfigmapTests=true
        stdSecretTests=true
        shift
        ;;
    -t | --test)
        testRun=true
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
k8s_e2e_perf_dir=${armada_perf_dir}/k8s-e2e-perf
k8s_e2e_perf_extra_args="--allowed-not-ready-nodes=1 --system-pods-startup-timeout=10s"

export GOPATH=${perf_dir}
export KUBE_ROOT="${GOPATH}"/src/k8s.io/kubernetes
export KUBECONFIG="/performance/config/${cluster_name}/kube-config-dal09-${cluster_name}.yml"
export KUBE_MASTER_IP=$(sed -n '/server:/p' "${KUBECONFIG}" | rev | cut -d/ -f1 | rev | head -1)
export KUBE_MASTER=$(echo ${KUBE_MASTER_IP} | cut -d: -f1)

cluster_config_dir="${perf_dir}"/config/${cluster_name}
export LOG_DUMP_SSH_KEY="${cluster_config_dir}"/.ssh/id_rsa
export LOG_DUMP_SSH_USER=root

cd "${KUBE_ROOT}"

k8s_output_dir="${KUBE_ROOT}"/_output
if [[ ! -d "${k8s_output_dir}" ]]; then
    printf "%s - Building Kubernetes....\n" "$(date +%T)"
    sudo -E make quick-release
    sudo chown -R jenkins:"Domain Users" "${KUBE_ROOT}" "${GOPATH}"/bin
    printf "%s - Kubernetes build completed.\n" "$(date +%T)"
fi

k8s_results_dir="${k8s_e2e_perf_dir}"/results
rm -rf "${k8s_results_dir}"
mkdir -p "${k8s_results_dir}"

e2eTestRun() {
    #function argument: K8s end to end test patterm e.g. "\[sig-scalability\]\sLoad.*\[Feature:Performance\]"
    k8s_e2e_test_pattern=$1

    echo
    echo Starting K8s performance test: ${_e2e_test_Pattern}
    echo

    test_start_time=$(date --iso-8601=seconds)
    sudo -E go run hack/e2e.go --get=false -- --check-version-skew=false --provider=skeleton --test --test_args="--ginkgo.dryRun=${testRun}  --ginkgo.focus=${k8s_e2e_test_pattern} --gather-metrics-at-teardown=false --allow-gathering-profiles=false --report-dir=${k8s_results_dir} --output-print-type=json ${k8s_e2e_perf_extra_args}" k8s_e2e_rc=$?
    test_end_time=$(date --iso-8601=seconds)

    sudo chown -R jenkins:"Domain Users" "${k8s_results_dir}"

}

if [ "${stdLoadTests}" = true ]; then
    echo
    echo Starting k8s Load tests
    e2eTestRun "\[sig-scalability\]\sLoad.*\[Feature:Performance\]"
fi

if [ "${stdDenityTests}" = true ]; then
    echo
    echo Starting k8s Density tests
    e2eTestRun "\[sig-scalability\]\sDensity.*\[Feature:Performance\]"
fi

if [ "${stdConfigmapTests}" = true ]; then
    echo
    echo Starting k8s Configmap tests
    e2eTestRun "\[sig-scalability\]\sLoad.*\[Feature:ManualPerformance\].*2\sconfigmaps"
fi

if [ "${stdSecretTests}" = true ]; then
    echo
    echo Starting k8s Secret tests :
    e2eTestRun "\[sig-scalability\]\sLoad.*\[Feature:ManualPerformance\].*2\ssecrets"
fi

echo
echo Finished
