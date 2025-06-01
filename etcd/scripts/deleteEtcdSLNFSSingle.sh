#!/bin/bash


kubectl delete -f etcd-slnfs-single.yml 
sleep 10
kubectl delete pvc data-etcd-slnfs-0
kubectl delete pvc data-etcd-slnfs-1
kubectl delete pvc data-etcd-slnfs-2
sleep 5
kubectl delete -f sl-persistent-volumes.yml 
kubectl get pods
kubectl get pvc
kubectl get pv
