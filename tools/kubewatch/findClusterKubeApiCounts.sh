#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Scan output of kubewatch, assumed to be in nohup.out, to find how many apiservers have been created for each cruiser

for i in `grep "namespace created" nohup.out | cut -d" " -f2 |cut -d"/" -f1`; do 
    echo -n "$i "
    grep $i nohup.out  |egrep "pod (created|deleted)"| grep kube-apiserver- | wc -l 
done > new.cluster.kube-api.txt
