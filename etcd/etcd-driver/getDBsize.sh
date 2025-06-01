#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
if [[ $# -ne 2 ]]; then
    echo "Usage: `basename $0` <endpoints> <total time to run for (mins)>"
    exit 1
fi

ENDPOINTS=$1
RUNTIME=$2
SLEEPTIME=10

NUMLOOPS=$(($RUNTIME/$SLEEPTIME + 1))

export ETCDCTL_API=3
echo "Running Get DB size for $NUMLOOPS loops"
for (( i=1; i<=$NUMLOOPS; i++))
do
   TIME=$(date +"%T")
   echo $TIME >> /perftest/etcd/etcd-driver/results/GetDBSize.txt
   /opt/bin/etcdctl $ETCDCREDS --endpoints $ENDPOINTS endpoint status >> /perftest/etcd/etcd-driver/results/GetDBSize.txt 2>&1
   sleep $(($SLEEPTIME))m
done
