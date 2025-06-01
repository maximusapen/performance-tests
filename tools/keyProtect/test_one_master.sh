#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

source ./test_conf.sh

clusterNum=1
cluster=kpcluster${clusterNum}
clusterId=bnj425020hks7uaham9g

# Set total number of restarts.  This is equivalent to number of DEKs to be added.
totalRestart=3

for i in $(seq ${secretStart} ${secretEnd}); do
    echo
    echo "*** Processing secret $i ***"
    # Restart master
    config/restart_one_master.sh ${clusterId}
    cd secret
    # Add one secret
    ./add_secrets.sh ${clusterNum} ${clusterNum} ${i} ${i}
    # Get all secrets
    ./get_secrets.sh ${clusterNum} ${clusterNum}
    cd ..
done
