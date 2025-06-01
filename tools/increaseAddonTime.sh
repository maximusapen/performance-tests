#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# Script to Increase the AddOnManager Interval (TEST_ADDON_CHECK_INTERVAL_SEC)
#
# Usage: increaseAddonTime.sh <Prefix to change time for> <new_time>

###########################################################################
# main
###########################################################################
set -x
if [[ $# -ne 2 ]]; then
    echo "Usage: `basename $0` <prefix> <time>"
    echo "Please supply the prefix to increase the time for, and the desired time"
    exit 1
fi
PREFIX=$1
TIME=$2

for i in `ls -d /mnt/nfs/$PREFIX*`
do
    echo $i; sudo sed -e "s/value: \"300\"/value: \"$TIME\"/g" $i/templates/master-deployment.yaml > `basename $i`.master-deployment.yaml
done

date
mkdir -p done
for i in `ls $PREFIX*master-deployment.yaml`
do
    echo $i; kubectl apply -f $i; mv $i done/
    sleep 10
done
date
