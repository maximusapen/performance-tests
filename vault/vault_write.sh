#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# Run vault_login.sh to login in vault before running this script
set -a
source vault_env.sh
set +a

if [[ $# -lt 2 ]]; then
    echo
    echo You need to pass in the key and key value to write to vault
    echo "Usage: vault_write.sh <key> <key value>"
    echo
    exit 1;
fi

key=$1
keyValue=$2

vault write ${VAULT_PATH}/${key} ${key}="${keyValue}"

writtenKey=$(./vault_list.sh | grep ${key})

if [[ ${writtenKey} == "" ]]; then
    echo
    echo Failed to write ${key} to Vault
    exit 1
fi

echo ${writtenKey} written to Vault
echo
echo Reminder:  Add the new key to vault_keylist for Jenkins jobs.
