#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

. etcd-perftest-config

mode=create

if [[ $1 == "delete" ]]; then
    mode=delete
    kubeDeleteParams=" --ignore-not-found"
fi

if [[ ! -f ${ETCDCLUSTER_NAME}-client-service-np.yml && ${ETCDCLUSTER_NAME} != "etcd-perftest" ]]; then
    if [[ ! -f etcd-perftest.yml || ! -f etcd-perftest.notls.yml || ! -f etcd-perftest-client-service-np.yml || ! -f etcd-perftest-client-service-lb.yml ]]; then
        echo "ERROR: etcd-perftest-*.yml files doesn't exist. Can't create new configuration files for ${ETCDCLUSTER_NAME}"
        exit 1
    fi
    echo "Configuring new yaml files for etcd cluster ${ETCDCLUSTER_NAME}"
    sed -e "s/etcd-perftest/${ETCDCLUSTER_NAME}/g" etcd-perftest.yml > ${ETCDCLUSTER_NAME}.yml
    sed -e "s/etcd-perftest/${ETCDCLUSTER_NAME}/g" etcd-perftest.notls.yml > ${ETCDCLUSTER_NAME}.notls.yml
    sed -e "s/etcd-perftest/${ETCDCLUSTER_NAME}/g" etcd-perftest-client-service-np.yml > ${ETCDCLUSTER_NAME}-client-service-np.yml
    sed -e "s/etcd-perftest/${ETCDCLUSTER_NAME}/g" etcd-perftest-client-service-lb.yml > ${ETCDCLUSTER_NAME}-client-service-lb.yml
    if [[ -f etcd-perftest-peer-tls.yml ]]; then
        sed -e "s/etcd-perftest/${ETCDCLUSTER_NAME}/g" etcd-perftest-peer-tls.yml > ${ETCDCLUSTER_NAME}-peer-tls.yml
    fi
    if [[ -f etcd-perftest-peer-tls.yml ]]; then
        sed -e "s/etcd-perftest/${ETCDCLUSTER_NAME}/g" etcd-perftest-server-tls.yml > ${ETCDCLUSTER_NAME}-server-tls.yml
    fi
    echo "The following configuration files have been created and can be modified before running this script again to create the cluster"
    ls -l ${ETCDCLUSTER_NAME}*.yml
    exit 0
fi 

if [ ${USE_CERTIFICATES} == true ]; then
    if [[ ${mode} == "create" ]]; then
        kubectl -n ${NAMESPACE} get secret ${ETCDCLUSTER_NAME}-server-tls 1>/dev/null 2>&1
        if [[ $? -ne 0 ]]; then
            ./generate-etcd-certs.sh
        fi
    else
        kubectl -n ${NAMESPACE} ${mode} secret ${ETCDCLUSTER_NAME}-server-tls ${kubeDeleteParams}
        kubectl -n ${NAMESPACE} ${mode} secret ${ETCDCLUSTER_NAME}-client-tls ${kubeDeleteParams}
        kubectl -n ${NAMESPACE} ${mode} secret ${ETCDCLUSTER_NAME}-peer-tls  ${kubeDeleteParams}
        kubectl -n ${NAMESPACE} ${mode} secret ${ETCDCLUSTER_NAME}-ca-tls ${kubeDeleteParams}
    fi
    kubectl -n ${NAMESPACE} ${mode} -f ${ETCDCLUSTER_NAME}.yml
    podCount=$(grep "size:" ${ETCDCLUSTER_NAME}.yml | awk '{print $2}')
else
    kubectl -n ${NAMESPACE} ${mode} -f ${ETCDCLUSTER_NAME}.notls.yml
    podCount=$(grep "size:" ${ETCDCLUSTER_NAME}.notls.yml | awk '{print $2}')
fi
kubectl -n ${NAMESPACE} ${mode} -f ${ETCDCLUSTER_NAME}-client-service-np.yml
kubectl -n ${NAMESPACE} ${mode} -f ${ETCDCLUSTER_NAME}-client-service-lb.yml

if [[ ${mode} == "create" ]]; then
    while (true); do
        ready=$(kubectl -n ${NAMESPACE} get pods -l etcd_cluster=${ETCDCLUSTER_NAME} | grep Running | wc -l)
        echo "Ready pods: ${ready}"
        if [[ ${ready} -eq ${podCount} ]]; then
            break
        fi
        sleep 20
    done
    source updateTestConfigFile.sh
    cp etcd-perftest-config ../etcd-driver/imageDeploy/
fi
