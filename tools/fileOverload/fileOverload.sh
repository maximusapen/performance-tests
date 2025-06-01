#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Use up all the file descriptors in the system

holdSeconds=600

if [[ $# -eq 1 ]]; then
    holdSeconds=$1
fi

echo "Holding for $holdSeconds seconds"

maxFilesPerfProcess=1000000
maxFileDescriptors=`cat /proc/sys/fs/file-max`
totalProcesses=$((maxFileDescriptors/maxFilesPerfProcess+1))

echo "Max file Descriptors: $maxFileDescriptors"
echo "Total processes: $totalProcesses"

echo "Start: " date
for (( i=0; i<$totalProcesses; i++ )); do
    sudo ./fileOverload --files $maxFilesPerfProcess --hold $holdSeconds --rLimit &
done

wait
echo "Done: " date
