#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Monitor the time to get services
# Usage: ./checkServiceTime.sh <cluster name> <duration in minutes>

CLUSTER=$1
MONITOR_MIN=$2

CYCLE_SLEEP_SEC=10
CYCLES=$((MONITOR_MIN*60/CYCLE_SLEEP_SEC))

. setPerfKubeconfig.sh $CLUSTER

echo "Monitoring $CLUSTER for $CYCLES $CYCLE_SLEEP_SEC second cycles for a total of $MONITOR_MIN minutes"
for ((i=0; i<CYCLES; i++)); do
	time kubectl get svc > /dev/null
	sleep $CYCLE_SLEEP_SEC
done
