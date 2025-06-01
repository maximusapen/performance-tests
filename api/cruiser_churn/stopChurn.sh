#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Run from /performance/stats/churn to stop cruiser_churn

GOPATH="/performance"

sudo pkill -SIGINT cruiser_churn

ps -aef | egrep "cruiser_churn"
echo
echo "Waiting for cruiser_churn to exit. Should complete within 30 minutes. .... "
CNT=$(ps -aef | egrep cruiser_churn | grep -v grep | wc -l)
while [[ $CNT -gt 0 ]]; do
    CNT=$(ps -aef | egrep cruiser_churn | grep -v grep | wc -l)
    echo -n "."
    sleep 30
done
echo
echo "Done waiting"
echo
ps -aef | egrep "cruiser_churn"
