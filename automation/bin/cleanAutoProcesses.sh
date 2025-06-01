#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2017, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

echo Requested to clean up Auto processses for cluster prefixes: $1
cluster_prefix_list=$(echo $1 | sed "s/,/ /g")
echo cluster_prefix_list: $cluster_prefix_list

killPids() {
    pids=$1
    echo $pids

    # Now kill the pid on the list
    for pid in ${pids[@]}; do
        echo killing $pid and descendants
        # Kill the pid and all its descendants
        sudo kill -9 $pid $(list_descendants $pid)
    done
}

# Get list of all descendant pids from the supplied pid
list_descendants () {
  local children=$(ps -o pid= --ppid "$1")

  for pid in $children
  do
    list_descendants "$pid"
  done

  echo "$children"
}

if [[ "${cluster_prefix_list}" == "" ]]; then
    # Clean up all Auto processes.
    echo "Cleaning up all Auto processes"
    pids=$(ps -ef | grep Auto | grep -v grep | grep -v cleanAutoProcesses | grep -v " vi " | awk '{print $2}')
    killPids "$pids"

    # Clean up all processes with kubeconfig using Perf cluster
    pids=$(ps -ef | grep "\--kubeconfig=/performance/config" | grep -v grep | awk '{print $2}')
    killPids "$pids"
else
    echo Cleaning up clusters: ${cluster_prefix_list}
    # Clean up Auto processes for each cluster listed
    for cluster_prefix in ${cluster_prefix_list}; do
        # Clean up all Auto processes for each matching cluster_prefix.
        echo "Cleaning up $cluster_prefix"
        ps -ef | grep Auto | grep $cluster_prefix | grep -v grep | grep -v cleanAutoProcesses
        pids=$(ps -ef | grep Auto | grep $cluster_prefix | grep -v grep | grep -v cleanAutoProcesses | awk '{print $2}')
        killPids "$pids"

        # Clean up all processes with reference to the cluster_prefix
        pids=$(ps -ef | grep ${cluster_prefix} | grep -v grep | awk '{print $2}')
        killPids "$pids"
    done
fi
