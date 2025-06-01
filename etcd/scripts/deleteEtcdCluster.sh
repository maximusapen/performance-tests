#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

kubectl delete -f etcd-cluster.yml 
sleep 10
kubectl delete pvc data-etcd-0
kubectl delete pvc data-etcd-1
kubectl delete pvc data-etcd-2
sleep 5
kubectl delete -f persistent-volumes.yml
kubectl get pods
kubectl get pvc
kubectl get pv
