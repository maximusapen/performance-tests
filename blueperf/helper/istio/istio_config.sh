# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019, 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Set up your ibmcloud ks to connect to carrier hosting your cluster
# Set up your KUBECONFIG before running this script
#export KUBECONFIG=<cruiser kube config file>

if [ $# -lt 2 ]; then
    echo "Usage: istio_config.sh < install | uninstall > < cluster name > < k8s version > [--extras]"
    echo "  Script to install/uninstall istio addon to cluster."
    echo "  Optional --extras to add istio-extras addon"

    exit 1
fi

perf_dir=/performance
action=$1
cluster_name=$2
k8s_version=$3
# Set highlight colour
yel=$'\e[1;33m'
end=$'\e[0m'

includeExtras=false
if [[ "$4" == "--extras" ]]; then
    includeExtras=true
fi

waitForAddonReady() {
    cluster_name=$1
    maxWaitTime=$2
    addon_name=$3

    curWaitTime=0

    addonReady=false
    pollingInterval=120

    while [[ ${curWaitTime} -lt ${maxWaitTime} ]]; do
        addonCheck=$(${perf_dir}/bin/armada-perf-client2 cluster addon ls --cluster "${cluster_name}" --json | jq -r ".[] | select(.name==\"${addon_name}\") | select (.healthStatus!= null) | select (.healthStatus|contains(\"Addon Ready\"))")

        if [[ -z ${addonCheck} ]]; then
            sleep ${pollingInterval}
            ((curWaitTime += ${pollingInterval}))
        else
            addonReady=true
            echo "${yel}Addon took $(($curWaitTime / 60)) minutes to become Ready${end}"
            break
        fi
    done

    if [[ "${addonReady}" != true ]]; then
        printf "\n%s - Gave up waiting for \"%s\" addon to be ready. Exiting.\n\n" "$(date +%T)" "${addon_name}"
        ${perf_dir}/bin/armada-perf-client2 cluster addon ls --cluster "${cluster_name}"
        exit 1
    fi
}

if [[ "${action}" == "install" ]]; then
    # Enable istio addon
    echo Enable istio addon for cluster "${cluster_name}"

    # Check kube major and minor versions and ignore patch version
    if [[ ${k8s_version} == "1.13"* || ${k8s_version} == "1.14"* || ${k8s_version} == "1.15"* ]]; then
        echo "Istio addon is only supported for Kubernetes versions v1.16+. Exiting."
        exit 1
    fi

    ibmcloud plugin list
    ${perf_dir}/bin/armada-perf-client2 cluster addon enable istio --cluster "${cluster_name}"
    ${perf_dir}/bin/armada-perf-client2 cluster addon ls --cluster "${cluster_name}"

    echo Waiting for up to 30 mins for istio addon to become ready
    waitForAddonReady "${cluster_name}" 1800 istio

    echo Creating Istio gateway, virtualservice and istio ingress
    kubectl apply -f istio/acmeair-gateway.yaml
    kubectl apply -f istio/acmeair-virtualservice.yaml
    # runAuto.sh generates acmeair-ingress.yaml from acmeair-ingress-template.yaml before running install with this script
    kubectl apply -f istio/acmeair-ingress.yaml

    if [[ ${includeExtras} == true ]]; then
        # Enable istio-extras addon
        echo Enable istio-extras addon for cluster "${cluster_name}"
        ${perf_dir}/bin/armada-perf-client2 cluster addon enable istio-extras --cluster "${cluster_name}"
        ${perf_dir}/bin/armada-perf-client2 cluster addon ls --cluster "${cluster_name}"

        echo Waiting for up to 30 mins for istio-extras addon to become ready
        waitForAddonReady "${cluster_name}" 1800 istio-extras
    fi

    # Use automated istio injection - acmeair pods run in default namespace
    # Alternatively, you can use manual sidecar injection but you need to install istioctl:
    #     kubectl apply -f <(istioctl kube-inject -f < deployment yaml>)
    kubectl label namespace default istio-injection=enabled --overwrite=true

    ${perf_dir}/bin/armada-perf-client2 cluster addon ls --cluster "${cluster_name}"
fi

if [[ "${action}" == "uninstall" ]]; then
    if [[ ${includeExtras} == true ]]; then
        echo Disable istio-extras addon for cluster "${cluster_name}"
        echo y | ${perf_dir}/bin/armada-perf-client2 cluster addon disable istio-extras --cluster "${cluster_name}"
        ${perf_dir}/bin/armada-perf-client2 cluster addon ls --cluster "${cluster_name}"
    fi

    echo Deleting Istio gateway, virtualservice and istio ingress
    kubectl delete --ignore-not-found=true -f istio/acmeair-gateway.yaml
    kubectl delete --ignore-not-found=true -f istio/acmeair-virtualservice.yaml
    kubectl delete --ignore-not-found=true -f istio/acmeair-ingress-template.yaml

    kubectl label namespace default istio-injection-

    echo Disable istio addon for cluster "${cluster_name}"
    echo y | ${perf_dir}/bin/armada-perf-client2 cluster addon disable istio --cluster "${cluster_name}"

    ${perf_dir}/bin/armada-perf-client2 cluster addon ls --cluster "${cluster_name}"
fi
