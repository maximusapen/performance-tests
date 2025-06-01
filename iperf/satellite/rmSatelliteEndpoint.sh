#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# Remove all satellite cloud or location endpoints with same prefix from location

if [[ $# -lt 2 ]]; then
    echo
    echo "Usage: ./rmSatelliteEndpoint.sh <endpointPrefix> <cloud | location>"
    echo
    exit 1
fi

endpointPrefix=$1
endpointType=$2

source envFile

echo Removing these satellite endpoints
ibmcloud sat endpoint ls --location ${location_id} | grep ${endpointPrefix} | grep ${endpointType}

endpoints=$(ibmcloud sat endpoint ls --location ${location_id} | grep ${endpointPrefix} | grep ${endpointType} | awk '{print $1}')

for endpoint in $endpoints; do
    echo Removing $endpoint
    echo yes | ibmcloud sat endpoint rm --location ${location_id} --endpoint ${endpoint}
done

echo ibmcloud sat endpoint ls --location ${location_id}
ibmcloud sat endpoint ls --location ${location_id}
