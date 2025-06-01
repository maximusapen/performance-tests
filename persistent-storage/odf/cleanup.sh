#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# This script is to cleanup any left over bits after deleting ODF Cluster
# It came from https://cloud.ibm.com/docs/openshift?topic=openshift-ocs-manage-deployment#ocs-rm-cleanup-resources

ocscluster_name=`oc get ocscluster | awk 'NR==2 {print $1}'`
oc delete ocscluster --all --wait=false
kubectl patch ocscluster/$ocscluster_name -p '{"metadata":{"finalizers":[]}}' --type=merge
oc delete ns openshift-storage --wait=false
sleep 20
kubectl -n openshift-storage patch persistentvolumeclaim/db-noobaa-db-0 -p '{"metadata":{"finalizers":[]}}' --type=merge
kubectl -n openshift-storage patch cephblockpool.ceph.rook.io/ocs-storagecluster-cephblockpool -p '{"metadata":{"finalizers":[]}}' --type=merge
kubectl -n openshift-storage patch cephcluster.ceph.rook.io/ocs-storagecluster-cephcluster -p '{"metadata":{"finalizers":[]}}' --type=merge
kubectl -n openshift-storage patch cephfilesystem.ceph.rook.io/ocs-storagecluster-cephfilesystem -p '{"metadata":{"finalizers":[]}}' --type=merge
kubectl -n openshift-storage patch cephobjectstore.ceph.rook.io/ocs-storagecluster-cephobjectstore -p '{"metadata":{"finalizers":[]}}' --type=merge
kubectl -n openshift-storage patch cephobjectstoreuser.ceph.rook.io/noobaa-ceph-objectstore-user -p '{"metadata":{"finalizers":[]}}' --type=merge
kubectl -n openshift-storage patch cephobjectstoreuser.ceph.rook.io/ocs-storagecluster-cephobjectstoreuser -p '{"metadata":{"finalizers":[]}}' --type=merge
kubectl -n openshift-storage patch noobaa/noobaa -p '{"metadata":{"finalizers":[]}}' --type=merge
kubectl -n openshift-storage patch backingstores.noobaa.io/noobaa-default-backing-store -p '{"metadata":{"finalizers":[]}}' --type=merge
kubectl -n openshift-storage patch bucketclasses.noobaa.io/noobaa-default-bucket-class -p '{"metadata":{"finalizers":[]}}' --type=merge
kubectl -n openshift-storage patch storagecluster.ocs.openshift.io/ocs-storagecluster -p '{"metadata":{"finalizers":[]}}' --type=merge
kubectl -n openshift-storage patch secrets rook-ceph-mon -p '{"metadata":{"finalizers": []}}' --type merge
kubectl -n openshift-storage patch configmap rook-ceph-mon-endpoints -p '{"metadata":{"finalizers": []}}' --type merge
kubectl -n openshift-storage patch cephobjectstoreuser.ceph.rook.io/prometheus-user -p '{"metadata":{"finalizers":[]}}' --type=merge
kubectl -n openshift-storage patch storagesystem.odf.openshift.io/ocs-storagecluster-storagesystem -p '{"metadata":{"finalizers":[]}}' --type=merge
sleep 20
oc delete pods -n openshift-storage --all --force --grace-period=0
sleep 20