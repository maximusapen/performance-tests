#!/bin/bash -ex
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# This script copies and executes the shell script to cruiser worker via sshdaemon

echo "Usage: ./exec-worker.sh < sshdaemon pod name > < shell script name >"
sshdaemon=$1
run_script=$2

echo "Copy script to sshdaemon"
kubectl cp ${run_script} ${sshdaemon}:/tmp/${run_script}
echo "Copy script from sshdaemon to cruiser worker"
kubectl exec ${sshdaemon} -it scp /tmp/${run_script} root@localhost:/tmp/${run_script}
echo "Run script on cruiser worker"
kubectl exec ${sshdaemon} -it ssh root@localhost /tmp/${run_script}
