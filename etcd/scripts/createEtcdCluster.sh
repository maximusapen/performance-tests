#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

sleep 10
kubectl create -f persistent-volumes.yml
sleep 10
kubectl get pv
kubectl create -f etcd-cluster.yml 
sleep 10
kubectl get pvc
kubectl get pods
