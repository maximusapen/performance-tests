#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to find all Node NotReady occurrences from controller-manager logs

if [[ $# -ne 1 ]]; then
    echo "Usage: `basename $0` <time_period> "
    echo "<time_period> = The time period to look in the logs for - e.g 1h = 1 hour"
    exit 1
fi
since=$1
DATE=$(date -u +"%FT%H%M")

# Get logs from controller-manager container
docker_id=$(sudo docker ps | grep -m 1 controller-manager | awk '{print $1}')
sudo docker logs $docker_id --since $since -t > rawLogs_$DATE.txt 2>&1

grep -a -E "NotReady as of|unresponsive as of" rawLogs_$DATE.txt > NotReadyLogs_$DATE.txt
grep -a -E "is healthy again" rawLogs_$DATE.txt > HealthyLogs_$DATE.txt

OIFS=$IFS
IFS=$'\n'

# Use this for mapping IP -> hostnames
hostnames=$(/opt/bin/calicoctl get nodes -o wide)

# Insures that jenkins jobs will complete successfully even if no outages
touch NotReadyResults_$DATE.txt

for line in `cat NotReadyLogs_$DATE.txt`; do
  time=$(echo $line | awk '{print $1}')
  host=$(echo $line | awk '{print $7}')
  hostname=$(echo "$hostnames" | grep -a -m 1 -w $host | awk '{print $1}' | cut -d$'.' -f1)
  not_ready_time="??"
  # Calculate how long the node was NotReady for
  for line2 in `grep -a "Node $host is healthy again" HealthyLogs_$DATE.txt`; do
    nr_secs=$(date -d $time +%s)
    r_time=$(echo $line2 | awk '{print $1}')
    r_secs=$(date -d $r_time +%s)
    # Look for first match where Node went Ready AFTER the time it went NotReady
    if [[ $r_secs -gt $nr_secs ]]; then
      not_ready_time=$(( $r_secs - $nr_secs ))
      break
    fi
  done
  # Write to results file
  time_no_ms=$(echo $time | cut -d$'.' -f1)
  echo "$time_no_ms, $not_ready_time, $host, $hostname" >> NotReadyResults_$DATE.txt
done

# Sort results by hostname
sort -k 4 -o NotReadyResults_$DATE.txt NotReadyResults_$DATE.txt

# Cleanup intermediate files
rm rawLogs_$DATE.txt
rm NotReadyLogs_$DATE.txt
rm HealthyLogs_$DATE.txt
