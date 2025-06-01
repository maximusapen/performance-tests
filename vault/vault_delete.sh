#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# Run vault_login.sh to login in vault before running this script
set -a
source vault_env.sh
set +a

if [[ $# -lt 1 ]]; then
    echo
    echo You need to pass in the key to be deleted
    echo "Usage: vault_delete.sh <key> "
    echo
    exit 1;
fi

key=$1

vault delete ${VAULT_PATH}/${key}
