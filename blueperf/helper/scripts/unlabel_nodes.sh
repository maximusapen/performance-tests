#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

labels="booking-db customer-db flight-db authservice customerservice flightservice bookingservice"

nodes=$(kubectl get node | grep -v NAME | awk '{print $1}')

echo Attempt unlabel all Acmeair labels for all nodes.
for node in $nodes; do
    echo Unlabel $node
    for label in $labels; do
        kubectl label node $node ${label}- >/dev/null
    done
done
