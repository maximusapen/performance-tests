#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Monitor meminfo on workers from Nate Rockwell

max_dirt=0

while (true); do
    date=$(date +"%Y-%m-%dT%H:%M:%S") 
    mem=$(cat /proc/meminfo | egrep "^(Dirty|Cache|MemFree)" | sort |awk '{print $2}' | paste -d " " - - -) 
    dirty=$(echo $mem | cut -d" " -f2) 
    if [[ $dirty -gt $max_dirt ]]; then 
        max_dirt=$dirty
    fi 
    echo $date $mem $max_dirt >> trackCache.log
    sleep 30
done
