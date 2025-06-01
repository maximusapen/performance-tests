#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Creates a timeline of cruiser creation based on the output from kubewatch

LOG=nohup.out

if [[ $# -lt 1 ]]; then
	echo "Usage: createTimeline.sh <cluster id> [<name of log file to search. Defaults to nohup.out>]"
fi
if [[ $# -gt 0 ]]; then
	CLUSTERID=$1
fi
if [[ $# -gt 1 ]]; then
	LOG=$2
fi
grep ${CLUSTERID} ${LOG} | egrep "(service|namespace) created|pod (created|updated Running)|deployment (created|updated [1-9])" | sed -e "s/master-${CLUSTERID}\///g" | sort -k 2 -u | awk -f createTimeline.awk | sort | sed -e "s/\([0-9]\)T\([0-9]\)/\1 \2/g"
