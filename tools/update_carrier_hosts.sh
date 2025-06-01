#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# This script will update the hosts file on a performance client machine, that has been built using:
#   https://alchemy-containers-jenkins.swg-devops.com/view/Armada-Performance/job/Armada-Performance/job/perfClient-setup/job/Build-and-Copy-perf-repo
# The hosts file will be updated with carrier machines associated with this performance client machine.  

network_source_root="/performance/alchemy-netint/network-source"
hosts_file="/etc/hosts"

echo "----------------------------------------------------------------"
echo "Updating ${hosts_file} on ${HOSTNAME}"
echo "----------------------------------------------------------------"

# USe the performance client hostname to deduce the carrier machines of interest 
IFS='-' read -r -a hostname_array <<< "${HOSTNAME}"

# environment: e.g. stage, prod
environment=${hostname_array[0]}

if [[ $environment = stage ]]; then
    account="531277"
elif [[ $environment = stgiks ]]; then
    account="1858147"
else
    echo "Unknown environment - expecting stage or stgiks"
    exit 1
fi
devices_file="${network_source_root}/softlayer-data/Acct${account}/devices.csv"

# region: e.g. dal, lon, wdc
region=${hostname_array[1]:0:3}

# carrier number: e.g. 2, 3, 4, 5
carrier_num=${hostname_array[2]: -1}

# Find all devices that match the carrier pattern, e.g. stage-dal09-carrier3
carrier_env="${environment}-${region}\d{2}-carrier${carrier_num}"
mapfile -t carrier_hosts < <( cat ${devices_file} | grep -P ${carrier_env} )

# For each matching device, add or replace entry in hosts file
for host in "${carrier_hosts[@]}"
do
  IFS="," read -r -a host_details <<< ${host}
  hn=${host_details[0]%%\.*}
  private_ip=${host_details[4]}

  if [[ -n ${private_ip} ]]; then
    sudo sed -i "/${hn}/d" ${hosts_file}

    host_entry="${private_ip}	${hn}"
    echo -n "Adding host: "
    echo "${host_entry}" | sudo tee -a ${hosts_file}
  fi
done
