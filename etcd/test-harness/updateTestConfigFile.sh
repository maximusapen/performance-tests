#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Pulls the service endpoints from kubernetes and updates etcd-perftest-config

. etcd-perftest-config

etcd_endpoint_lb=$(kubectl -n armada get svc ${ETCDCLUSTER_NAME}-client-service-lb -o=jsonpath='{.status.loadBalancer.ingress[*].ip}{":"}{.spec.ports[*].nodePort}{"\n"}')
echo "Etcd lb endpoint: $etcd_endpoint_lb"
sed -i -e "s/^ETCD_LB_ENDPOINTS=.*$/ETCD_LB_ENDPOINTS=${etcd_endpoint_lb}/g" etcd-perftest-config

node_port=$(kubectl -n armada get svc ${ETCDCLUSTER_NAME}-client-service-np -o=jsonpath='{.spec.ports[*].nodePort}{"\n"}')
node=$(kubectl get nodes --no-headers | head -1 | cut -d" " -f1)
etcd_endpoint_np="${node}:${node_port}"
echo "Etcd nodeport endpoint: $etcd_endpoint_np"
sed -i -e "s/^ETCD_NP_ENDPOINTS=.*$/ETCD_NP_ENDPOINTS=${etcd_endpoint_np}/g" etcd-perftest-config
sed -i -e "s/cloud.ibm.com:[0-9]*/cloud.ibm.com:$node_port/g" etcd-perftest-config
