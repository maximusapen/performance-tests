#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Delete stranded IAM IDs
# Notes
# 1) Care must be taken to insure that all carriers in use by the performance team are listed below.
# 2) Run on version 0.3.112 or later of the container-service/kubernetes-service plugin

carriers="0 2 3 4 5 6"
minimumClusters=5
today=$(date "+%Y-%m-%d")

# Get list of service IDs
echo "Retrieving a list of the service IDs"
ibmcloud iam service-ids | grep -v "$today" > service-ids.txt

# Get list of clusters in each carrier
echo "Retrieving a list of clusters in each carrier"
rm -f carrier*-clusters.txt
for i in $carriers; do
    echo "    carrier$i"
    if [[ $i -eq 0 ]]; then
        ibmcloud ks api https://containers.test.cloud.ibm.com > /dev/null
    else
        ibmcloud ks api https://stage-us-south$i.containers.test.cloud.ibm.com > /dev/null
    fi
    if [[ $? -ne 0 ]]; then
        echo "ERROR: Couldn't set API for carrier$i"
        exit 1
    fi
    ibmcloud ks clusters > carrier$i-clusters.txt
    if [[ $? -ne 0 ]]; then
        echo "ERROR: Failed to get a list of clusters on carrier$i"
        exit 1
    fi
    # Make sure there are some clusters in the carrier (i.e. error on the safe side)
    clusters=$(egrep "Dallas|dal09|dal10|dal12|dal13" carrier$i-clusters.txt | wc -l)
    if [[ $i -gt 0 && $clusters -lt $minimumClusters ]]; then
        echo "ERROR: Carrier$i has $clusters clusters which has less than the minimum of $minimumClusters."
        exit 1
    fi
done

# Generate a list of stranded service IDs
echo "Finding the stranded IDs"
rm -f stranded-service-ids.txt 
for i in `grep cluster service-ids.txt | awk '{print $2}' | cut -d"-" -f2`; do 
    grep $i carrier* > /dev/null
    if [[ $? -ne 0 ]]; then
        grep $i service-ids.txt >> stranded-service-ids.txt
    fi
done

# Delete a range of the stranded IDs
if [[ -f stranded-service-ids.txt ]]; then
    echo "Deleting the stranded IDs"
    for i in `cat stranded-service-ids.txt | awk '{print $1}' | sort -u`; do
        ibmcloud iam service-id-delete $i -f
    done
else
    echo "There aren't any stranded IDs to be deleted"
fi
