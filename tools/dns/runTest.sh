#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

params=$1
query=$2
reportDir=$3

# Existing run-* will cause the mv command below to fail, so renaming to backup-run-*
cd out
runReports=$(ls -d run-*)
for runReport in ${runReports}; do
    echo "Moving ${runReport} to backup-${runReport}"
    mv ${runReport} backup-${runReport}
done
unlink latest
cd ..

python3 py/run_perf.py --params params/${params}/${query}.yaml --out-dir out --use-cluster-dns

mv out/run-* out/${reportDir}/${params}-${query}
unlink out/latest
