#!/bin/bash
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
