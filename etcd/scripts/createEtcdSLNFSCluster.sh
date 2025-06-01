#!/bin/bash

sleep 10
kubectl create -f sl-persistent-volumes.yml 
sleep 10
kubectl get pv
kubectl create -f etcd-slnfs-cluster.yml 
sleep 10
kubectl get pvc
kubectl get pods
