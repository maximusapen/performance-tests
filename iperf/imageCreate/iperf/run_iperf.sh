#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018,2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# By default run in server mode - but if the -c argument is passed, then just use the arguments supplied
client_mode=false
for var in "$@"
do
    if [[ $var == "-c" ]]; then
      client_mode=true
    fi
done

if [[ $client_mode == true ]]; then
    args=${*}
else
    args="-s ${*}"
fi

echo "args = $args"
iperf3 $args
