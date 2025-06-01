#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright Maximus Apen, 2025 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Retries the docker versions from each perf client.

for i in `grep "stage-dal[0-9]*-perf[1-5]-client-[0-9]*" /etc/hosts | awk '{print $2}'`;  do
    echo $i
    ssh $i sudo docker version | egrep "Version:|Client:|Engine:|containerd:|runc:|docker-init:"
done
