#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Run on carrier master which has the armada-data to set test BOM to all masters
# Test BOMs usually has a lower minor version so it won't be used for default BOM.
# This means you can't use API to update the BOM version as downgrade is not allowed.
#
# If only KP clusters exists on carrier, you can use openvpnserver pods to get the KP cluster list
clusterIds=$(kubectl get pod -n kubx-masters | grep openvpnserver | sed "s/-/ /g" | awk '{print $2}')

# Search for 5/5 master pods if non-KP clusters also exists on carrier

# Set the test BOM version.  E.g.
# testBOM="1.15.6_1258"
testBOM="< test BOM version >"
date
echo "Setting testBOM ${testBOM} on all clusters"
for clusterId in ${clusterIds}; do
    echo Processing ${clusterId}
    armada-data set Master -field DesiredAnsibleBomVersion -value ${testBOM} -pathvar MasterID=test-${clusterId}-000001
    sleep 1
done
date
