#!/bin/bash -e

# This script can run in the following scenarios:
# - In Jenkins jobs where vault role id and secret id credentials are set up by conductors
#   for crn_v1_staging_public_containers-kubernetes_us-south_armada-performance-read.
# - With vault role id and secret from Thycotic.

if [ $# -lt 2 ]; then
    echo
    echo You need to pass in vault role id and vault secret id as following usage:
    echo "Usage:"
    echo "    vault_getsecret.sh <vault_role_id_value> <vault_secret_id_value>"
    echo "For testing with test_secrets>"
    echo "    vault_getsecret.sh test_role_id test_secret_id"
    echo
    exit 1
fi

vault_role_id_value=$1
vault_secret_id_value=$2

if [[ ${vault_role_id_value} == "test_role_id" ]]; then
    isTest=true
    vaultDir=.
    set -a
    source ${vaultDir}/vault_env.sh
    source ${vaultDir}/test_secrets
    set +a
    # Setting test value here for STAGE_GLOBAL_ARMPERF_VSI_PRIVATEKEY and STAGE_GLOBAL_ARMPERF_VSI_PUBLICKEY
    key_with_space="ssh-rsa VALUE_STAGE_GLOBAL_ARMPERF_VSI_PUBLICKEY  armada.performance@uk.ibm.com"
    key_with_eol="-----BEGIN OPENSSH PRIVATE KEY-----          # pragma: allowlist secret
VALUE_STAGE_GLOBAL_ARMPERF_VSI_PRIVATEKEY
-----END OPENSSH PRIVATE KEY-----"
else
    isTest=false
    vaultDir=$(find . -name vault)
    set -a
    source ${vaultDir}/vault_env.sh
    set +a

    # This is not a test. Set up vault environment
    # Install vault
    wget -nv https://releases.hashicorp.com/vault/0.10.4/vault_0.10.4_linux_amd64.zip
    unzip vault_0.10.4_linux_amd64.zip
    chmod +x vault
    sudo cp vault /usr/local/bin
    ls -l /usr/local/bin/vault
    which vault

    # Set up vault environment for appRole
    # vault_role_id_value and vault_secret_id_value environment variable for the vault needs to be set up
    export VAULT_HOST=vserv-us.sos.ibm.com
    export VAULT_ADDR=https://${VAULT_HOST}:8200
    vaultPath=generic/crn/v1/staging/public/containers-kubernetes/us-south/-/-/-/-/stage/armada-performance/

    # Set up vault token for appRole read
    vaultToken=$(vault write auth/approle/login role_id=${vault_role_id_value} secret_id=${vault_secret_id_value} | grep token | grep -v token_ | awk '{print $2}')
    if [[ ${vaultToken} == "" ]]; then
        echo
        echo Failed to set up vault token
        exit 1
    fi
fi

# Add key secret to file
add_secret() {
    # SECRET_FILE location set up in vault_env.sh
    if [[ ${isTest} == true ]]; then
        if [[ $1 == "STAGE_GLOBAL_ARMPERF_VSI_PRIVATEKEY" ]]; then
            # Test using key_with_eol
            val=${key_with_eol}
        elif [[ $1 == "STAGE_GLOBAL_ARMPERF_VSI_PUBLICKEY" ]]; then
            # Test using key_with_space
            val=${key_with_space}
        else
            val=$(grep $1 test_secrets | sed "s/$1=//")
        fi
    else
        # Key names in Vault could have Date appended to them - so figure out which name to read - we want to get the most recent, which will be the last to match the grep
        key_to_read=$(echo "${vault_key_names}" | grep $1 | tail -1)
        echo "key_to_read is ${key_to_read}"
        val=$(VAULT_TOKEN=${vaultToken} vault read -field=${key_to_read} ${vaultPath}/${key_to_read} | sed "s/"${key_to_read}"[ ]*//")
    fi

    if [[ ${val} == *' '* || ${val} == *'\n'* ]]; then
        # Special handling if secret value has space or eol when writing to file
        export $1="$val"
        if [[ ${writeSecret} == true ]]; then
            echo $1=\""$val"\" >>${SECRET_FILE}
        fi
    else
        export $1=${val}
        if [[ ${writeSecret} == true ]]; then
            echo $1=$val >>${SECRET_FILE}
        fi
    fi

}

if [[ ${SECRET_FILE} == "" ]]; then
    writeSecret=false
else
    writeSecret=true
    : >${SECRET_FILE}
fi
echo writeSecret: ${writeSecret}

# Parsed vault_keylist file to get key list in vault
keys=$(cat ${vaultDir}/vault_keylist | grep -v "# Message:" | awk '{ print $1 }')

# Temporary workaround in place until https://github.ibm.com/alchemy-conductors/team/issues/10352 is resolved.
#vault_key_names=$(VAULT_TOKEN=${vaultToken} vault list ${vaultPath})
vault_key_names=$(cat ${vaultDir}/keysInVault.txt)


# Add all key values to
for key in ${keys}; do
    add_secret $key
done
vaultKeys=$(echo ${keys} | sed "s/ /,/g")

# Check that secret file has secrets
if [[ ${armada_performance_db_password} == "" ]]; then
    echo
    echo Failed to get secret from vault
    exit 1
fi

if [[ ${SECRET_FILE} != "" ]]; then
    echo Secrets written to ${SECRET_FILE}
fi
