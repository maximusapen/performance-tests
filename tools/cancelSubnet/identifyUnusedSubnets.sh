#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

privateVlanID=2263901
publicVlanID=2263903

if [[ $1 == "private" ]]; then
    vlanID=${privateVlanID}
elif [[ $1 == "public" ]]; then
    vlanID=${publicVlanID}
else
    printf "Please specify \"private\" or \"public\".\n"
    exit 1
fi

perf_dir=/performance
armada_perf_dir=${perf_dir}/armada-perf
export GOPATH=${perf_dir}

# Copy current perf.toml file so that we can restore later
cp ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml ${armada_perf_dir}/armada-perf-client2/config/orig-perf-metadata.toml

declare -A subnetMap

printf "Subnet use on VLAN %s\n\n" "${vlanID}"

for carrierNum in 4 5; do
    printf "Checking Carrier %d for subnet usage\n" "${carrierNum}"
    rsync ${armada_perf_dir}/armada-perf-client2/config/carrier${carrierNum}_stage-perf-metadata.toml ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml
    subnets=$(${perf_dir}/bin/armada-perf-client2 subnets --json | jq -r ".[] | select(.vlan_id==\"${vlanID}\") | [.id,.properties.display_label,.properties.subnet_type,.properties.bound_cluster]|@tsv")

    while read subnet; do
        id=$(echo ${subnet} | awk '{print $1}')
        displayLabel=$(echo ${subnet} | awk '{print $2}')
        subnetType=$(echo ${subnet} | awk '{print $3}')
        boundCluster=$(echo ${subnet} | awk '{print $4}')

        key=$(printf "%-7s - %s" "${id}" "${displayLabel}")
        if [[ ${boundCluster} != "" ]]; then
            name=$(${perf_dir}/bin/armada-perf-client2 cluster get --cluster "${boundCluster}" --json | jq -r .name)
            cluster="${name} (${boundCluster})"
            subnetMap[${key}]=${cluster}
        else
            if [[ ${subnetMap[${key}]+_} == "" ]]; then
                if [[ ${subnetType} == *"primary"* ]]; then
                    subnetMap[${key}]='PRIMARY'
                else
                    subnetMap[${key}]='NOT IN USE'
                fi
            fi
        fi

    done <<<"${subnets}"
done

printf "\n%-9s %-17s   %s\n%s\n" "ID" "DISPLAY LABEL" "STATUS" "--        -------------       ------"
for s in "${!subnetMap[@]}"; do
    printf "%-27s : %s\n" "${s}" "${subnetMap[$s]}"
done

# Restore original perf-metadata.toml
mv ${armada_perf_dir}/armada-perf-client2/config/orig-perf-metadata.toml ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml
