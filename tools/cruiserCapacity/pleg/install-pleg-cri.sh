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

key=$(grep client-key $KUBECONFIG | awk '{print $2}')
certificate=$(grep client-certificate $KUBECONFIG | awk '{print $2}')

for worker in ${workers[@]}
do
    echo $worker

    ssh root@$worker mkdir pleg
    ssh root@$worker mkdir mem
    ssh root@$worker mkdir cert
    scp pleg-cri.sh root@$worker:~/pleg/pleg.sh
    scp meminfo.sh root@$worker:~/mem
    scp get-kubelet-go.sh root@$worker:~/cert
    scp $key root@$worker:~/cert
    scp $certificate root@$worker:~/cert

    # Need to ssh into worker to start process
    #ssh root@$1 cd pleg; nohup ./pleg.sh &; cd ../mem; nohup ./meminfo.sh &
done

