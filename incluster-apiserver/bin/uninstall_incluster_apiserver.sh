#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

#Defaults
get_namespace="incluster-apiserver-target"
namespace=default

perf_dir=/performance
armada_perf_dir=${perf_dir}/armada-perf
helm_dir=/usr/local/bin

while test $# -gt 0; do
    case "$1" in
    -h | --help)
        echo "uninstall_incluster_apiserver.sh - uninstall kubernetes incluster apiserver test resources"
        echo " "
        echo "uninstall_incluster_apiserver.sh [options]"
        echo " "
        echo "options:"
        echo "-h, --help                        show brief help"
        echo "-n, --namespace k8s_namespace     kubernetes namespace for deployment"
        echo "-i, --instances                   Number of instances to be uninstalled"
        echo "-m, --get_namespace               The namespace used in the test for the get requests"
        exit 0
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
    -i | --instances)
        shift
        if test $# -gt 0; then
            instances=$1
        else
            echo "instances number not specified"
            exit 1
        fi
        shift
        ;;
    -m | --get_namespace)
        shift
        if test $# -gt 0; then
            get_namespace=$1
        else
            echo "get_namespace value not specified"
            exit 1
        fi
        shift
        ;;
    *)
        args="${args} "$*
        break
        ;;
    esac
done

# Uninstall the instances
for ((i = 1; i <= ${instances}; i = i + 1)); do
    helm uninstall "incluster-apiserver-${i}" --namespace ${namespace}
done
kubectl delete namespace ${namespace}

# Uninstall the target
for ((i = 1; i <= ${instances}; i = i + 1)); do
    ns=${get_namespace}${i}
    helm uninstall "incluster-apiserver-target" --namespace ${ns}
    kubectl delete namespace ${ns}
done
