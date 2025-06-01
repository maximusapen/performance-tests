#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Set up your KUBECONFIG before running this script
#export KUBECONFIG=<cruiser kube config file>

# Run on all workers in cluster
workers=$(kubectl get node | grep -v NAME | awk '{print $1}')

# Alternatively only run on selected workers as in below example.
#workers=("10.143.222.139" "10.143.222.208" "10.143.222.223" "10.143.222.225" "10.143.222.231")

kubectl top node >> topnode.log 2>&1

for worker in ${workers[@]}
do
    echo Getting pleg data from $worker
    scp root@$worker:~/pleg/nohup.out $worker.pleg.log
    scp root@$worker:~/pleg/pleg.docker.lst $worker.pleg.docker.lst
    scp root@$worker:~/mem/trackCache.log $worker.trackCache.log
    ssh root@$worker /root/cert/get-kubelet-go.sh > $worker.kubelet.go.log
    ssh root@$worker cat /proc/meminfo > $worker.meminfo
done
