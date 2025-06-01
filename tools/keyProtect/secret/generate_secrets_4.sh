#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Generate 500 secrets in yaml directory

# *** Note that you need to modify secret.yaml with genuine secret data first. ***
# See README.md

# Modify following parameters to generate secrets required
clusterStart=1
clusterEnd=950
secretStart=1
secretEnd=20

mkdir -p yaml

date
for i in $(seq ${clusterStart} ${clusterEnd}); do
    for j in $(seq ${secretStart} ${secretEnd}); do
        secretYaml=yaml/perf-${i}-secret-${j}.yaml
        sed "s/CLUSTERID/${i}/" secret.yaml | sed "s/SECRETID/${j}/" >${secretYaml}
        chmod 664 ${secretYaml}
        echo Generated ${secretYaml}
    done
done
date
