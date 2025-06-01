#!/bin/bash -x
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Multiple of these scripts will create etcd-operator clusters in parellel
# create_operatpr_clusters.sh sets up the conditions under which this script runs.

if [[ $# -ne 3 ]]; then
    echo "Usage: `basename $0` <thread> <clusters> <prefix>"
    exit 1
fi

cd ${WORKSPACE}/etcd-operator

THREAD=$1
CLUSTERS=$2
PREFIX=$3

if [[ -z $NAMESPACE ]]; then
    NAMESPACE="etcd-operator"
fi

SLEEP_INTERVAL=10
CREATE_TIMEOUT=1200

for (( i=1; i<=$CLUSTERS; i=i+1 )); do
     cd ${WORKSPACE}/etcd-operator/example/tls/certs
     mkdir $PREFIX$i-certs
     cp * $PREFIX$i-certs
     cd $PREFIX$i-certs
     rm -rf *.crt
     rm -rf *.key
     mv server.json server.json.orig
     sed "s/example/$PREFIX$i-etcd/g" server.json.orig > server.json
     sed -i "s/.default./.$NAMESPACE./g" server.json
     mv peer.json peer.json.orig
     sed "s/example/$PREFIX$i-etcd/g" peer.json.orig > peer.json
     sed -i "s/.default./.$NAMESPACE./g" peer.json

     ./gen-cert.sh
     kubectl -n $NAMESPACE create secret generic $PREFIX$i-etcd-peer-tls --from-file=peer-ca.crt --from-file=peer.crt --from-file=peer.key
     kubectl -n $NAMESPACE create secret generic $PREFIX$i-etcd-server-tls --from-file=server-ca.crt --from-file=server.crt --from-file=server.key
     kubectl -n $NAMESPACE create secret generic $PREFIX$i-etcd-client-tls --from-file=etcd-client-ca.crt --from-file=etcd-client.crt --from-file=etcd-client.key

     cd ${WORKSPACE}/etcd-operator
     cp ${WORKSPACE}/armada-performance/etcd/etcd-operator/config/example-tls-cluster.yaml example/tls/$PREFIX$i-tls-cluster.yaml
     sed -i "s/etcd-/$PREFIX$i-etcd-/g" example/tls/$PREFIX$i-tls-cluster.yaml
     sed -i "s/example/$PREFIX$i-etcd/g" example/tls/$PREFIX$i-tls-cluster.yaml
     SECONDS=0
     createTime=0
     kubectl create -f example/tls/$PREFIX$i-tls-cluster.yaml -n $NAMESPACE
     expectedPodsRunning=$(grep "size:" example/tls/$PREFIX$i-tls-cluster.yaml | tr -d '[:space:]' | cut -d$':' -f2)
     numPodsRunning=$(kubectl get pods -n $NAMESPACE --no-headers | grep Running | grep $PREFIX$i | wc -l)
     while (( numPodsRunning != expectedPodsRunning && createTime < CREATE_TIMEOUT )); do
            sleep $SLEEP_INTERVAL
            numPodsRunning=$(kubectl get pods -n $NAMESPACE --no-headers | grep Running | grep $PREFIX$i | wc -l)
            echo "Pods running: $numPodsRunning , Expected pods Running: " $expectedPodsRunning
            createTime=$SECONDS
     done
     if ((numPodsRunning == expectedPodsRunning)); then
       STATUS=0
       echo "Create succeeded for $PREFIX$i in $createTime seconds"
     else
       STATUS=1
       echo "Create failed/timed out for $PREFIX$i in $CREATE_TIMEOUT seconds"
     fi

     # Deal with reality that metrics db has 30 second granularity.
     if [[ $createTime -lt 30 ]]; then
         sleep $((30-$createTime))
     fi
     if [[ $STATUS -eq 0 ]]; then
         echo "CreateTime: $createTime"
         for (( j=1; j<=5; j++ )); do
             set +x
             curl -XPOST --header "X-Auth-User-Token: apikey $METRICS_KEY" -d "[{\"name\" : \"dal09.performance.${CARRIER}_stage.EtcdOperatorCluster.thread$THREAD.Create_Time.sparse-avg\",\"value\" : $createTime}]" https://metrics.stage1.ng.bluemix.net/v1/metrics
             curl -XPOST --header "X-Auth-User-Token: apikey $METRICS_KEY" -d "[{\"name\" : \"dal09.performance.${CARRIER}_stage.EtcdOperatorCluster.thread$THREAD.Create_Success.count\",\"value\" : 1}]" https://metrics.stage1.ng.bluemix.net/v1/metrics
             set -x
             if [[ $? -eq 0 ]]; then
                 echo "Posted cluster create in $createTime at `date +"%Y%m%d_%H%M%S"`"
                 break
             fi
             echo "Metric post $j failed"
             sleep $j
         done
     else
         echo "Deployment FAILED with return code of $STATUS in $createTime seconds"
         echo "$PREFIX$i" >> ${WORKSPACE}/EtcdOperatorClusterCreateFailures.txt
         for (( j=1; j<=5; j++ )); do
             set +x
             curl -XPOST --header "X-Auth-User-Token: apikey $METRICS_KEY" -d "[{\"name\" : \"dal09.performance.${CARRIER}_stage.EtcdOperatorCluster.thread$THREAD.Create_Failed.count\",\"value\" : 1}]" https://metrics.stage1.ng.bluemix.net/v1/metrics
             set -x
             if [[ $? -eq 0 ]]; then
                 echo "Posted failed cluster create `date +"%Y%m%d_%H%M%S"`"
                 break
             fi
             echo "Metric post $j failed"
             sleep $j
         done
     fi
done
