#!/bin/bash


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
