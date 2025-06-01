#!/bin/bash -ex
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Extract CARRIER, ENV and REALENV from OVERRIDE_HOST parameter to
# property file carrier.properties

properties_file="carrier.properties"

# Save original IFS before setting delimiter to "-"
OIFS=$IFS
IFS="-"

# Split OVERRIDE_HOST into array based on "-" delimiter
env_carrier=($OVERRIDE_HOST)

# Reset IFS to original value
IFS=$OIFS

# Set REALENV and ENV
REALENV="${env_carrier[0]}-${env_carrier[1]}"

ENV=$REALENV
if [ $REALENV == "dev-mex01" ]; then
    ENV="dev-mon01"
fi

# Set CARRIER
CARRIER=${env_carrier[2]}

# Write to properties file to inject into Jenkins
echo "ENV=$ENV" > $properties_file
echo "REALENV=$REALENV" >> $properties_file
echo "CARRIER=$CARRIER" >> $properties_file
