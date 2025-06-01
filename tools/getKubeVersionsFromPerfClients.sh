#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Retries the kube client and server versions from each perf client.

for i in `grep "stage-dal[0-9]*-perf[1-5]-client-[0-9]*" /etc/hosts | awk '{print $2}'`;  do
    echo $i
    carrier=$(echo $i | sed "s/stage-dal[0-9]*-perf\([0-9]\)-client-.*/\1/g")
    if [[ $carrier -eq 1 ]]; then
        carrier=5
    fi
    ssh $i KUBECONFIG=/performance/config/carrier${carrier}_stage/admin-kubeconfig kubectl version | grep version | awk '{print $1, $5}'
done
