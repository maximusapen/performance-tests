#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Fixes routes when Cisco AnyConnect messes with stage VPN setup
# If there are any parameters then it tells you what it will do and exits before doing anything

# Assumptions: tunnel in question is the only tunnel that goes to the '10.x.x.x.' network"

if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "ERROR: Must run on Mac OS since it hasn't been tested elsewhere."
    exit 1
fi

for i in `ifconfig | grep "^utun" | cut -d: -f1`; do 
    ifconfig $i | grep "inet 10"
    if [[ $? -eq 0 ]]; then
        tunnel=$i
        break
    fi
done

if [[ -z $tunnel ]]; then
    echo "ERROR: Couldn't figure out which tunnel VPN is using"
    exit 1
fi

badRoutes=$(netstat -nr | grep "^10\." | grep -v $tunnel | wc -l | awk '{print $1}')

if [[ $badRoutes -eq 0 ]]; then
    echo "INFO: Didn't find any bad routes"
    exit 0
fi

if [[ $# -ge 1 ]]; then
    echo "VPN tunnel: $tunnel"
    echo "VPN bad routes: $badRoutes"
    echo "Some example routes to be fixed"
    netstat -nr | grep "^10\." | grep -v $tunnel | head -4
    exit 0
fi

# Fix routes
for network in `netstat -nr | grep "^10\." | grep -v $tunnel | awk '{ print $1 }'`; do
    echo "Changing $network to route to $tunnel"
    sudo route change $network -interface $tunnel
done
