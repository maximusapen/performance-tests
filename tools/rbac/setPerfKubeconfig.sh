#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Sets KUBECONFIG for a cluster
# Usage: . setPerfKubeconfig.sh <cluster name or id>

configsDir=/performance/config
iksConfigsDir="${HOME}/.bluemix/plugins/container-service/clusters"

CLUSTER=$1

function getKubeConfig {
        if [[ -n $(which ibmcloud) ]]; then
            LAST_KUBECONFIG="$KUBECONFIG"
            export KUBECONFIG="$configAdminPath/admin-kubeconfig"
            echo $KUBECONFIG
            ibmcloud ks cluster config --cluster ${CLUSTER} ${ADMIN_CONTEXT}
            if [[ $? -eq 0 ]]; then
                echo "KUBECONFIG=${KUBECONFIG}"
                if [[ $1 ]]; then
                    kubectl config get-contexts
                fi
            else
                echo "ERROR: Config directory not found and config couldn't be retrieved via ibmcloud: ${configPath}"
                export KUBECONFIG="$LAST_KUBECONFIG"
            fi
        else
            echo "ERROR: Config directory not found: ${configPath}"
        fi
}

if [[ $# -eq 0 ]]; then
    echo "Current setting: KUBECONFIG=${KUBECONFIG}"
else
    ADMIN_CONTEXT="--admin"
    echo $2
    if [[ $# -eq 2 && $2 == "user" ]]; then
        ADMIN_CONTEXT=""
    fi

    configDir=${CLUSTER}
    configPath=${configsDir}/${configDir}
    configAdminPath=${configsDir}/${configDir}-admin
    configIksAdminPath=${iksConfigsDir}/${configDir}-admin
    configIksPath=${iksConfigsDir}/${configDir}-admin

    if [[ -d ${configAdminPath} ]]; then
        configPath=${configAdminPath}
    elif [[ -d ${configIksAdminPath} ]]; then
        configPath=${configIksAdminPath}
    elif [[ -d ${configIksPath} ]]; then
        configPath=${configIksPath}
    fi

    if [[ -d ${configPath} ]]; then
        config=$(ls ${configPath}/kube-config-*.yml 2>/dev/null)
        if [[ -f ${config} ]]; then
            export KUBECONFIG=${config}
            echo "KUBECONFIG=${KUBECONFIG}"
        elif [[ -f ${configPath}/admin-kubeconfig ]]; then
            export KUBECONFIG=${configPath}/admin-kubeconfig
            echo "KUBECONFIG=${KUBECONFIG}"
        else
            config=$(ls ${configPath}/*.yml 2>/dev/null)
            if [[ -f ${config} ]]; then
                export KUBECONFIG=${config}
                echo "KUBECONFIG=${KUBECONFIG}"
            else
                echo "ERROR: Couldn't find configuration file"
            fi
        fi
        # This assumes only 2 contexts in config file. Good most of the time
        if [[ -z ${ADMIN_CONTEXT} ]]; then
            echo "Use user context"
            CONTEXT=$(kubectl config get-contexts --no-headers -o name | grep ${CLUSTER} | grep -v admin)
            if [[ -z ${CONTEXT} ]]; then
                getKubeConfig
                CONTEXT=$(kubectl config get-contexts --no-headers -o name | grep ${CLUSTER} | grep -v admin)
            fi
        else
            CONTEXT=$(kubectl config get-contexts --no-headers -o name | egrep "${CLUSTER}.*/admin")
            if [[ -z ${CONTEXT} ]]; then
                getKubeConfig
                CONTEXT=$(kubectl config get-contexts --no-headers -o name | egrep "${CLUSTER}.*/admin")
            fi
        fi
        echo "Context: $CONTEXT"
        kubectl config use-context ${CONTEXT}
        kubectl config get-contexts
    else
        getKubeConfig true
    fi
fi
