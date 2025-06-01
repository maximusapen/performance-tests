#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

#Defaults
address=""
namespace="default"
pods=1
id=1
load_balancer=false
lb_postfix=""
prefix="prefix"

while test $# -gt 0; do
    case "$1" in
        -h|--help)
            echo "netperfclient.sh - runs Linux iperf client on existing netperf pods"
            echo " "
            echo "netperfclient.sh [options] args"
            echo " "
            echo "options:"
            echo "-h, --help                        show brief help"
            echo "-n, --namespace k8s_namespace     kubernetes namespace for deployment"
            echo "-i, --id identifier               specify an identifier to use (allows multiple deployments in 1 cluster) - must be an integer"
            echo "-a, --address                     The IP address of the iperfserver to connect to"
            echo "-p, --prefix                      The prefix the results file name under ./results/"
            echo "-l, --loadbalancer                use the load balancer"
            exit 0
            ;;
        -i|--id)
            shift
            if test $# -gt 0; then
                if ! [[ "$1" =~ ^[0-9]+$ ]]
                then
                    echo "ID must be an integer"
                    exit 1
                fi
                id=$1
            else
                echo "ID not specified"
                exit 1
            fi
            shift
            ;;
        -a|--address)
            shift
            if test $# -gt 0; then
                address=$1
            else
                echo "Address not specified"
                exit 1
            fi
            shift
            ;;
        -n|--namespace)
            shift
            if test $# -gt 0; then
                namespace=$1
            else
                echo "Namespace not specified"
                exit 1
            fi
            shift
            ;;
        -p|--prefix)
            shift
            if test $# -gt 0; then
                prefix=$1
            else
                echo "Prefix not specified"
                exit 1
            fi
            shift
            ;;
        -l|--loadbalancer)
            load_balancer=true
            lb_postfix="-lb"
            shift
            ;;
        *)
            args="${args} "$*
            break
            ;;
    esac
done

echo Namespace: ${namespace}
if [ "$load_balancer" = true ]; then
    let "port = ${id} + 40520"
else
    let "port = ${id} + 30520"
fi
echo Port: ${port}
echo id: ${id}

if [[ -n ${address} ]]; then
    echo Address: ${address}
else
    echo "-a or --address must be specified"
    exit 1
fi

# This is a client - so always set -c & port
args="${args} -c ${address} -p ${port}"
echo Args: ${args}

# Some of the pods may have been deleted so netperf-pod${id} may not exist.
# Instead get the ${id}th pod that does exist
pod=$(kubectl -n ${namespace} get pod --no-headers | grep "netperf-pod[0-9]*${lb_postfix} " | awk '{print $1}' | sed -n "${id}p" )
echo "Using pod ${pod}"

kubectl -n ${namespace} exec ${pod} -- iperf3 ${args} > results/${prefix}_${pod}
