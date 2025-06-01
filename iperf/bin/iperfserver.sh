#!/bin/bash

#Defaults
registry=""
environment=""
namespace="default"
pods=1
id=1
load_balancer=false
#public_vlan="2917088"

deploymentName=iperfserver

while test $# -gt 0; do
    case "$1" in
    -h | --help)
        echo "iperfserver.sh - runs kubernetes based Linux iperf server"
        echo " "
        echo "iperfserver.sh [options] args"
        echo " "
        echo "options:"
        echo "-h, --help                        show brief help"
        echo "-e, --environment environment     Registry environment namespace (e.g dev7, stage1, etc.)"
        echo "-g, --registry registry_url       image registry location"
        echo "-d, --deployment chart_name       helm chart deployment name"
        echo "-n, --namespace k8s_namespace     kubernetes namespace for deployment"
        echo "-p, --pods PODS                   specify number of replicas(pods) within which tests are to be run"
        echo "-i, --id identifier               specify an identifier to use (allows multiple deployments in 1 cluster) - must be an integer"
        echo "-l, --loadbalancer                use the load balancer"
        echo "-vl, --vlan public_VLAN           public_VLAN (mandatory for ROKS cluster only)"
        exit 0
        ;;
    -p | --pods)
        shift
        if test $# -gt 0; then
            pods=$1
        else
            echo "Number of pods not specified"
            exit 1
        fi
        shift
        ;;
    -i | --id)
        shift
        if test $# -gt 0; then
            if ! [[ "$1" =~ ^[0-9]+$ ]]; then
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
    -d | --deployment)
        shift
        if test $# -gt 0; then
            deploymentName=$1
        else
            echo "Helm chart deployment name not specified"
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
    -l | --loadbalancer)
        load_balancer=true
        shift
        ;;
    -vl | --vlan)
        shift
        if test $# -gt 0; then
            public_vlan=$1
        else
            echo "public VLAN not specified"
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

echo Chart: ${deploymentName}
echo Namespace: ${namespace}
echo Pods: ${pods}
let "port = ${id} + 30520"
let "lb_port = ${id} + 40520"
echo Port: ${port}
echo id: ${id}
echo publicVLAN: ${public_vlan}
echo Args: ${args}

if [[ -n ${registry} ]]; then
    echo Registry: ${registry}
    setRegistry="--set image.registry=${registry}"
fi

if [[ -n ${environment} ]]; then
    echo Registry Environment: ${environment}
    setEnvironment="--set image.name=armada_performance_${environment}/iperf"
fi

# Delete any existing job from previous runs
helm uninstall "${deploymentName}-${id}" --namespace ${namespace} 2>/dev/null

# Create a new config map so that we can pass in the supplied parameters to the pod creation
kubectl delete configmap "${deploymentName}-${id}-config" --ignore-not-found=true --namespace=${namespace}
sleep 40
kubectl create configmap "${deploymentName}-${id}-config" --namespace=${namespace} --from-literal=PERF_IPERF_ARGS="${args}"

# Use the first public VLAN and zone we find - need to iterate through as some nodes may not have a public VLAN
OIFS=$IFS
IFS=$'\n'
for node in $(kubectl get nodes --no-headers); do
    node_name=$(echo ${node} | awk '{print $1}')
    vlan=$(kubectl get node $node_name -o=jsonpath='{.metadata.labels.publicVLAN}')
    zone=$(kubectl get node $node_name -o=jsonpath='{.metadata.labels.failure-domain\.beta\.kubernetes\.io/zone}')
    if [[ ! -z $vlan ]]; then
        # Found the vlan
        echo "Using public VLAN $vlan from node $node_name"
        break
    fi
done
IFS=$OIFS

if [[ -z ${vlan} ]]; then
    # This is likely a ROKS cluster with no publicVLAN data available in nodes.  Need to pass data as argument to this script
    vlan=${public_vlan}
    if [[ -z ${vlan} ]]; then
        echo "Unable to get a Public VLAN from any node - so Load Balancer will not work."
        echo "If this is a ROKS cluster, pass in public VLAN with --vlans argument"
        exit 1
    fi
    echo "Using vlan ${vlan} from --vlan ${public_vlan}"
fi

# Use helm to install the chart and execute the tests as a kubernetes job on the cluster
helm install "${deploymentName}-${id}" ../imageDeploy/iperfserver --namespace=${namespace} --set podCount=${pods} --set port=${port} --set id=${id} --set lb.port=${lb_port} --set lb.zone=${zone} --set lb.vlanID=\"${vlan}\" ${setCPU} ${setVMBytes} ${setRegistry} ${setEnvironment}
