#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to call kubectl get nodes but sort output by hostname
# It requires access to kubectl for the carrier & either calicoctl (so run on the master)
# or a /etc/hosts file containing the carrier worker hosts


OIFS=$IFS
IFS=$'\n'

hostnames=""
if [ -f "/opt/bin/calicoctl" ]; then
   for line in `/opt/bin/calicoctl get nodes -o wide`; do
     name=$(echo "$line" | awk '{print $1}' | cut -d$'.' -f1)
     ip=$(echo "$line" | awk '{print $3}')
     hostnames+="$ip $name"$'\n'
   done
else
   hostnames=$(cat /etc/hosts)
fi

host_cnt=$(echo "$hostnames" | egrep "worker|master" | wc -l)

result=""
for line in `kubectl get nodes`; do
  ip_address=$(echo $line | awk '{print $1}')
  if [ $ip_address == "NAME" ]; then
    # Header line
    if [ $host_cnt -gt 1 ]; then
        echo "HOSTNAME                           "$line
    else
        echo "   "$line
    fi
    continue
  fi

  host=$(echo $line | awk '{print $7}')
  hostname=$(echo "$hostnames" | grep -a -m 1 -w $ip_address | awk '{print $2}' | cut -d$'.' -f1)

  result+="$hostname   $line"$'\n'
done

# Sort results by hostname
echo -n "$result" | sort
