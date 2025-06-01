#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019, 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

perf_dir=/performance
armada_perf_dir=${perf_dir}/armada-perf

for type in public private; do
    # Identify the unbound subnets
    unusedSubnets=$(${armada_perf_dir}/tools/cancelSubnet/identifyUnusedSubnets.sh ${type} | grep "NOT IN USE" | awk '{print $3}')

    numUnused=$(echo $unusedSubnets | sed '/^\s*$/d' | wc -l)

    # Now delete the unbound subnets
    if [ "$numUnused" -ne "0" ]; then
        while read unusedSubnet; do
            echo "deleting subnet ${unusedSubnet} type ${type}"
            # Output 'yes' to the prompt from the command
            yes yes | /performance/bin/cancelSubnet -subnet=${unusedSubnet} -${type}
        done <<<"${unusedSubnets}"
    else
        printf "No subnets of type ${type} to delete\n"
    fi
done
