#!/bin/bash

#Defaults
registry=""
environment=""
namespace="default"
pods=1
cpu=""
vm=""
vmBytes="256M"

deploymentName=stress

while test $# -gt 0; do
        case "$1" in
                -h|--help)
                        echo "stress-driver - runs kubernetes based Linux stress utility"
                        echo " "
                        echo "stress-driver [options] args"
                        echo " "
                        echo "options:"
                        echo "-h, --help                  	show brief help"
                        echo "-c, --cpu N                       spawn N workers spinning on sqrt()"
                        echo "-m, --vm N                        spawn N workers spinning on malloc()/free()"
                        echo "--vm-bytes B                      malloc B bytes per vm worker (default is 256MB)"
                        echo "-e, --environment environment     Registry environment namespace (e.g dev7, stage1, etc.)"
                        echo "-g, --registry registry_url 	image registry location"
                        echo "-d, --deployment chart_name       helm chart deployment name"
                        echo "-n, --namespace k8s_namespace     kubernetes namespace for deployment"
                        echo "-p, --pods PODS             	specify number of replicas(pods) within which tests are to be run"
                        exit 0
                        ;;
                -c|--cpu)
                        shift
                        if test $# -gt 0; then
                                cpu=$1
                                args="${args}-c ${cpu} "
                        else
                                echo "Number of workers (CPU cores) not specified"
                                exit 1
                        fi
                        shift
                        ;;
                -m|--vm)
                        shift
                        if test $# -gt 0; then
                                vm=$1
                                args="${args}-m ${vm} "
                        else
                                echo "Number of workers (memory) not specified"
                                exit 1
                        fi
                        shift
                        ;;
                --vm-bytes)
                        shift
                        if test $# -gt 0; then
                                vmBytes=$1
                                args="${args}--vm-bytes ${vmBytes} "
                        else
                                echo "Number of bytes not specified"
                                exit 1
                        fi
                        shift
                        ;;
                -p|--pods)
                        shift
                        if test $# -gt 0; then
                                pods=$1
                        else
                                echo "Number of pods not specified"
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
                -d|--deployment)
                        shift
                        if test $# -gt 0; then
                                deploymentName=$1
                        else
                                echo "Helm chart deployment name not specified"
                                exit 1
                        fi
                        shift
                        ;;
                -e|--environment)
                        shift
                        if test $# -gt 0; then
                                environment=$1
                        else
                                echo "registry namespace environment not specified"
                                exit 1
                        fi
                        shift
                        ;;
                -g|--registry)
                        shift
                        if test $# -gt 0; then
                                registry=$1
                        else
                                echo "registry url not specified"
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
echo Args: ${args}

if [[ -n ${cpu} ]]; then
  echo CPU: ${cpu}
  setCPU="--set cpu=${cpu}" # CPU in milli-cores for Kubernetes resources
fi

if [[ -n ${vm} ]]; then
  echo MEM: "${vm} : ${vmBytes}"
  setVMBytes="--set vmBytes=${vmBytes}"
fi

if [[ -n ${registry} ]]; then
  echo Registry: ${registry}
  setRegistry="--set image.registry=${registry}"
fi

if [[ -n ${environment} ]]; then
  echo Registry Environment: ${environment}
  setEnvironment="--set image.name=armada_performance_${environment}/stress"
fi

# Delete any existing job from previous runs
helm uninstall ${deploymentName} --namespace=${namespace} 2> /dev/null

# Create a new config map so that we can pass in the supplied parameters to the pod creation
kubectl delete configmap "${deploymentName}-config" --ignore-not-found=true --namespace=${namespace}
kubectl create configmap "${deploymentName}-config" --namespace=${namespace} --from-literal=PERF_STRESS_ARGS="${args}" 

# Use helm to install the chart and execute the tests as a kubernetes job on the cluster
helm install ${deploymentName} ../imageDeploy/stress --namespace=${namespace} --set podCount=${pods} ${setCPU} ${setVMBytes} ${setRegistry} ${setEnvironment}
