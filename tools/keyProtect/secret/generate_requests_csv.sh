#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# For APIServer Load test with API calls to get many secrets.
# Generate requests.csv used in APIServer Load test.

# Backup the /performance/armada-perf/k8s-apiserver/requests.csv
# and replace with the generated requests.csv from this script.

# Modify following parameters to generate secrets required
# APIServer load test usually for one cluster so setting
# clusterStart and clusterEnd the same.
# Possibility of testing multiple clusters in one test
clusterStart=1
clusterEnd=1
secretStart=1
secretEnd=500

echo "Removing requests.csv if exists"
set +e
rm requests.csv
set -e

echo "Generating requests ..."
for i in $(seq ${clusterStart} ${clusterEnd}); do
    for j in $(seq ${secretStart} ${secretEnd}); do
        echo "GET,/api/v1/namespaces/default/secrets/perf-${i}-secret-${j}," >>requests.csv
    done
done
echo "Generated requests in requests.csv"
