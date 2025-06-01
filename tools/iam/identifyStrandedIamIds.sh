#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Delete stranded IAM IDs
#
# Script will just list service IDs that will be deleted unless '-delete' is passed in as parameter
# Usage: identifyStrandedIamIds.sh [-delete]
# [-delete] = Optional flag - if set delete the stranded service IDs - otherwise just list them
# It requires the STAGE_GLOBAL_ARMPERF_IBMCLOUD_APIKEY ENV Variable to be set before running the script

if [[ $1 == "-delete" ]]; then
    delete=true
    echo "Found '-delete' parameter so will delete any stranded IAM service IDs found."
else
    delete=false
    echo "Will list stranded IAM service IDs. If you want to delete them then call this script with '-delete' argument"
fi

perf_dir=/performance
armada_perf_dir=${perf_dir}/armada-perf
export GOPATH=${perf_dir}

API_TARGET="https://test.cloud.ibm.com"

# Copy current perf-metadata.toml file so that we can restore later
cp ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml ${armada_perf_dir}/armada-perf-client2/config/orig_perf-metadata.toml

minimumClusters=1
today=$(date "+%Y-%m-%d")

# Get list of service IDs
echo "Retrieving a list of the service IDs"
export IBMCLOUD_API_KEY=${STAGE_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}
export ARMADA_PERFORMANCE_API_KEY=${STAGE_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}
ibmcloud config --check-version=false
ibmcloud login -a $API_TARGET --no-region
ibmcloud iam service-ids | grep -v "$today" > service-ids.txt

# Get list of clusters in each carrier
echo "Retrieving a list of clusters in each carrier"
rm -f carrier*-clusters.txt
for carrierNum in 4 5; do
    printf "Listing clusters from Carrier %d \n" "${carrierNum}"
    rsync ${armada_perf_dir}/armada-perf-client2/config/carrier${carrierNum}_stage-perf-metadata.toml ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml
    ${perf_dir}/bin/armada-perf-client2 cluster ls > carrier${carrierNum}-clusters.txt

    if [[ $? -ne 0 ]]; then
        echo "ERROR: Failed to get a list of clusters on carrier${carrierNum}"
        mv ${armada_perf_dir}/armada-perf-client2/config/orig_perf-metadata.toml ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml
        exit 1
    fi
    # Make sure there are some clusters in the carrier (i.e. err on the safe side)
    clusters=$(${perf_dir}/bin/armada-perf-client2 cluster ls --json | jq .[].id | wc -l)
    if [[ $clusters -lt $minimumClusters ]]; then
        echo "ERROR: Carrier${carrierNum} has $clusters clusters which has less than the minimum of $minimumClusters."
        mv ${armada_perf_dir}/armada-perf-client2/config/orig_perf-metadata.toml ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml
        exit 1
    fi
done

# Get list of Satellite locations in each carrier
echo "Retrieving a list of Satellite locations in each carrier"
rm -f carrier*-locations.txt
for carrierNum in 0; do
    printf "Listing locations from Carrier %d \n" "${carrierNum}"
    rsync ${armada_perf_dir}/armada-perf-client2/config/satellite${carrierNum}_stage-perf-metadata.toml ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml
    ${perf_dir}/bin/armada-perf-client2 sat location ls > carrier${carrierNum}-locations.txt

    if [[ $? -ne 0 ]]; then
        echo "ERROR: Failed to get a list of locations on carrier${carrierNum}"
        mv ${armada_perf_dir}/armada-perf-client2/config/orig_perf-metadata.toml ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml
        exit 1
    fi
done

# Generate a list of stranded service IDs
echo "Finding the stranded IDs"
rm -f stranded-service-ids.txt
for i in `grep "cluster-" service-ids.txt | awk '{print $2}' | cut -d"-" -f2`; do
    grep -- "$i" carrier* > /dev/null
    if [[ $? -ne 0 ]]; then
        grep -- "$i" service-ids.txt >> stranded-service-ids.txt
    fi
done
# Look for Satellite locations which have a slightly different structure
# Note, this cleans up based on Sat location name rather than ID as that is what is stored in the service ID name
# This means if a Location exists with the same name as one which leaked service IDs previously they wouldn't get
# cleaned up - but this should be good enough, as they should eventually get cleaned up
for i in `grep "satellite-location-" service-ids.txt | awk '{print $2}' | cut -d"-" -f 3-`; do
    grep -- "$i" carrier* > /dev/null
    if [[ $? -ne 0 ]]; then
        grep -- "$i" service-ids.txt >> stranded-service-ids.txt
    fi
done

# Delete a range of the stranded IDs
if [[ -f stranded-service-ids.txt ]]; then
    echo "Listing/Deleting the stranded IDs"
    for i in `cat stranded-service-ids.txt | awk '{print $1}' | sort -u`; do
        if [[ $delete == "true" ]]; then
            echo "Deleting stranded IAM service ID: $i"
            ibmcloud iam service-id-delete $i -f
        else
            echo "Found stranded IAM service ID: $i"
        fi
        grep -- "$i" stranded-service-ids.txt
    done
else
    echo "There aren't any stranded IDs to be deleted"
fi

# Restore original perf.toml
mv ${armada_perf_dir}/armada-perf-client2/config/orig_perf-metadata.toml ${armada_perf_dir}/armada-perf-client2/config/perf-metadata.toml
