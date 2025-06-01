#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# This script tests the performance of RBAC authentication for admina and non-admin users.
# The cluster specified must have been previously loaded with the desired number of users.
# To use this script do the following:
#     nohup ./getEndUserAuthorizationTimes.sh <cluster name>
#     ./parseEndUserAuthorizationTimes.sh nohup.out
# The reason is that the version of `time` on the perf clients can't output to a file,
# so to record the result nohup is needed.

REQUEST_CNT=11

if [[ $# -ne 1 ]]; then
	echo "ERROR: Must specify cluster"
	exit 1
fi
CLUSTER=$1

function timeRequests {
	for ((i=0; i<${REQUEST_CNT}; i++)); do
		time kubectl get svc
	done
}

. setPerfKubeconfig.sh ${CLUSTER}
timeRequests

. setPerfKubeconfig.sh ${CLUSTER} user
timeRequests
