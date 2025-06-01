#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2017, 2023 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
helm_dir=/usr/local/bin
perftest=registry
perf_dir=/performance
armada_perf_dir=${perf_dir}/armada-perf
influxdb_metrics_dir=${perf_dir}/metrics
export GOPATH=${perf_dir}
export IKS_BETA_VERSION=1
export METRICS_DB_KEY="${armada_performance_db_password}"

# point to a temporary kubeconfig
export KUBECONFIG=$(mktemp).yml
touch $KUBECONFIG
# and make sure it gets cleaned up on exit
function cleanupconfig() {
  rm -f $KUBECONFIG
}
trap cleanupconfig EXIT

cd /performance/armada-perf/registry/bin
ibmcloud plugin install container-service -r "IBM Cloud" -f
ibmcloud plugin install container-registry -r "IBM Cloud" -f

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

wait_for_completion() {
  # This test runs in a cluster so it cannot write directly to Influxdb, so metrics are written to a file. Get the metrics files from the job with the name and namespace specified. Only wait for the results for the specified number of minutes.
  rm -rf ${influxdb_metrics_dir}/${perftest}*.json 2>/dev/null
  sleep 120
  # Wait for results
  getInfluxdbMetricsFromJob "perf-${perftest}-job" "${perftest}" 120

  printf "\n ----------- Historic logs from job ----------- \n"
  kubectl logs job/"perf-${perftest}-job" -n "${perftest}"
  printf "\n ---------------------------------------------- \n\n"
}

export PROD_GLOBAL_ARMPERF_REGDEV_APIKEY
export IBMCLOUD_API_KEY=${PROD_GLOBAL_ARMPERF_REGDEV_APIKEY}
export PROD_GLOBAL_ARMPERF_IBMCLOUD_APIKEY

# First run in each region using local registry
for region in ap-north us-south eu-gb eu-de ap-south; do
  ibmcloud login -a https://cloud.ibm.com -c 93f968a6f3fb265d95b62413c86c2e28 --no-region

  clustername="regtest-${region}"
  ibmcloud ks cluster config --cluster $clustername

  # Log into regional registry
  ibmcloud cr region-set $region
  ibmcloud cr login

  # Setup registry access
  source ${armada_perf_dir}/automation/bin/setupRegistryAccess.sh default true true
  source ${armada_perf_dir}/automation/bin/setupRegistryAccess.sh "${perftest}" true true

  # Run test
  ./perf-registry.sh -v -m -e "stgiks5" -n "registry" -g "stg.icr.io" -r "registry.$region.icr.io" -y $hyperkubeImage

  # Wait for job completion (max test time 1 hour)
  wait_for_completion

  # Next run in each region using the International/Global registry

  # New section for Registry testing the au_syd new cluster - clone of global registry
  if [[ "$region" == "ap-south" ]]; then
    ./perf-registry.sh -v -m -i -e "stgiks5" -n "registry" -g "stg.icr.io" -r "registry.$region.icr.io" -dns "168.1.26.2:icr.io"

    # Wait for job completion (max test time 1 hour)
    wait_for_completion
  fi
  # End of new section

  ./perf-registry.sh -v -m -i -e "stgiks5" -n "registry" -g "stg.icr.io" -r "registry.$region.icr.io"

  # Wait for job completion (max test time 1 hour)
  wait_for_completion

done
printf "\n%s - Test(s) Completed\n" "$(date +%T)"
