#!/bin/bash -e
#

# This script can run in the following scenarios:
# - With real secrets generated from vault_getsecret.sh
#       - ./generate_perf_toml.sh
# - With fake secrets using test_secrets file for testing
#       - ./generate_perf_toml.sh test_secrets

if [[ $# -eq 1 ]]; then
    echo "Secret file changed from \"${SECRET_FILE}\" to $1"
    SECRET_FILE=$1
fi

if [[ "${SECRET_FILE}" != "" ]]; then
    # Add secrets to environment variables, otherwise set up secrets environment variables
    # with vault_getsecret.sh before calling this script
    set -a
    source ${SECRET_FILE}
    set +a
fi

# Create directory for generated toml file
TOML_DIR="/tmp/armada_perf_toml"
mkdir -p ${TOML_DIR}
set +e
# Clean up any files in directory if already exists
rm -fr ${TOML_DIR}/*
set -e

# Get all toml templates and process common keys in format of KEY_<key>_KEY in template
for TEMPLATE in template/*.toml; do
    TEMPLATE_FILE=$(basename ${TEMPLATE})
    echo "Processing template ${TEMPLATE}"

    if [[ ${TEMPLATE} =~ "satellite"* ]]; then
        SATELLITE=true
    else
        SATELLITE=false
    fi

    # Determine the environment (e.g. stage, production)
    IFS="-" read -ra template_file_array <<<"${TEMPLATE_FILE}"
    TEMPLATE_ENV=${template_file_array[0]}

    # Ensure we start with an enpty file
    : >/tmp/tmp_$TEMPLATE_FILE

    # Process each line in the template file in turn
    while read line; do
        IFS='=' read -ra line_array <<<"${line}"
        template_key_name=${line_array[0]}
        template_key_value=${line_array[1]%%#*} # Strip off any comments

        case "${template_key_value}" in
        # Normal Keys
        *KEY_*_KEY*)
            # Strip off KEY prefix and suffix
            template_key_value=${template_key_value//KEY_/}
            template_key_value=${template_key_value//_KEY/}

            envvar_name="$(echo -e "${template_key_value}" | tr -d '[:space:] \"')"
            key_value="${!envvar_name}"

            printf '%s= \"%s\"\n' "${template_key_name}" "${key_value}" >>/tmp/tmp_$TEMPLATE_FILE
            ;;
        # Encrypted Keys
        *ENCKEY_*_ENCKEY*)
            # Strip off ENCKEY prefix and suffix
            template_key_value=${template_key_value//ENCKEY_/}
            template_key_value=${template_key_value//_ENCKEY/}

            envvar_name="$(echo -e "${template_key_value}" | tr -d '[:space:] \"')"

            export STAGE_GLOBAL_ARMPERF_CRYPTOKEY
            encrypted_key_value=$(${GOPATH}/bin/crypto -encrypt "${!envvar_name}")

            printf '%s= \"%s\"\n' "${template_key_name}" "${encrypted_key_value}" >>/tmp/tmp_$TEMPLATE_FILE
            ;;
        *)
            printf '%s\n' "${line}" >>/tmp/tmp_$TEMPLATE_FILE
            ;;
        esac
    done <${TEMPLATE}

    # Get carrier-specific keys
    if [ ${SATELLITE} == true ]; then
        CARRIER_KEY_LIST=$(cat vault_env.sh | grep -v "#" | grep "^satellite" | cut -d "=" -f 1)
    else
        CARRIER_KEY_LIST=$(cat vault_env.sh | grep -v "#" | grep "^${TEMPLATE_ENV}_carrier" | cut -d "=" -f 1)
    fi

    # Process carrier-specific environment variables in the format of in format "CARRIER_key_CARRIER"
    while IFS= read -r CARRIER_KEY; do
        echo $CARRIER_KEY

        IFS='_' read -ra carrier_key_array <<<"${CARRIER_KEY}"

        CARRIER_ENV=${carrier_key_array[0]}
        if [ ${SATELLITE} == true ]; then
            CARRIER_ID=${carrier_key_array[0]}
            CARRIER="${CARRIER_ID}"
        else
            CARRIER_ID=${carrier_key_array[1]}
            CARRIER="${CARRIER_ENV}_${CARRIER_ID}"
        fi
        KEY=$(echo ${CARRIER_KEY} | sed "s/"${CARRIER}_"//")

        if [ ${SATELLITE} == true ]; then
            TOML_FILE="${CARRIER_ID}_stage${TEMPLATE_FILE#"satellite"}"
        else
            TOML_FILE="${CARRIER_ID}_${TEMPLATE_FILE}"
        fi

        echo "Processing ${KEY} for ${CARRIER} to /tmp/${TOML_FILE}"

        if [[ ! -f /tmp/${TOML_FILE} ]]; then
            # Carrier toml file not created when processing the first key for the carrier
            echo "Creating /tmp/${TOML_FILE}"
            cp /tmp/tmp_${TEMPLATE_FILE} /tmp/${TOML_FILE}
        fi

        cat /tmp/${TOML_FILE} | sed "s|"CARRIER_${KEY}_CARRIER"|$(env | grep "${CARRIER_KEY}=" | sed "s|${CARRIER_KEY}=||")|" >/tmp/tmp_${TOML_FILE}
        cp /tmp/tmp_${TOML_FILE} /tmp/${TOML_FILE}

        # Move generated toml file to TOML_DIR
        cp /tmp/${TOML_FILE} ${TOML_DIR}
    done <<<"${CARRIER_KEY_LIST}"
    rm /tmp/${TOML_FILE}
done

echo
echo "Generated carrier toml file(s) in ${TOML_DIR}"
ls -l ${TOML_DIR}
