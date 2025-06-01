#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# Set up your vault environment
set -a
source vault_env.sh
set +a

if [ $# -lt 1 ]; then
    echo
    echo you can also pass in your GIT personal access token as follows:
    echo "    Usage: vault_login.sh <GIT_PERSONAL_ACCESS_TOKEN>"
    echo
    echo See https://help.github.com/en/articles/creating-a-personal-access-token-for-the-command-line
    echo to create your GIT personal access token.
    echo
    echo You will be prompted to pass in your GIT personal access token to login into Vault.
    echo

    vault login -method=github
else
    GIT_PERSONAL_ACCESS_TOKEN=$1
    vault login -method=github token=${GIT_PERSONAL_ACCESS_TOKEN}
fi

echo Login succeeded and the vault token is added to $HOME/.vault-token
