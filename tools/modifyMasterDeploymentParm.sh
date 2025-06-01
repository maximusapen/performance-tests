#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# Script to modify a parameter in the master-deployment.yaml for a cruiser,
# and apply the change
#
# Usage: modifyMasterDeploymentParm.sh <Prefix modify> <old_value> <new_value>

###########################################################################
# main
###########################################################################
set -x
if [[ $# -ne 3 ]]; then
    echo "Usage: `basename $0` <prefix> <old_value> <new_value>"
    echo "Please supply the master prefix to modify with the old_value & the new_value"
    echo "old_value & new_value should be strings that will be used in a sed expression"
    exit 1
fi
PREFIX=$1
OLDVALUE=$2
NEWVALUE=$3

echo "Changing ${OLDVALUE} to ${NEWVALUE} for all masters that match the prefix ${PREFIX}"

for i in `ls -d /mnt/nfs/$PREFIX*`
do
    echo $i; sudo sed -e "s/${OLDVALUE}/${NEWVALUE}/g" $i/templates/master-deployment.yaml > `basename $i`.master-deployment.yaml
done

date
mkdir -p done
for i in `ls $PREFIX*master-deployment.yaml`
do
    echo $i; kubectl apply -f $i; mv $i done/
    sleep 10
done
date
