#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Loads the specified number of RBAC users into the cluster specified by the current kubeconfig.
# USAGE: ./createRBACUserObjects.sh [<number of users:defaults to 5000>]

USERS=$1
if [[ ${USERS} == "" ]]; then
	USERS=5000
fi
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
date
#for i in {1..5000}; do
for (( i=0; i<${USERS}; i++ )); do
	export USERNUM=$i 
	cat <<EOF | kubectl apply -f -
apiVersion: ibm.com/v1alpha1
kind: RBACSync
metadata:
  name: user$USERNUM
spec:
  subject: IAM#user$USERNUM@ibm.com
  authzBinding:
    "*":
    - crn:v1:bluemix:public:iam::::serviceRole:Reader
    default:
    - crn:v1:bluemix:public:iam::::serviceRole:Writer
    kube-system:
    - crn:v1:bluemix:public:iam::::serviceRole:Manager
EOF
done
date
echo "=============================="
