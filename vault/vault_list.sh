#!/bin/bash -e

# Run vault_login.sh to login in vault before running this script
set -a
source vault_env.sh
set +a

vault list ${VAULT_PATH}
