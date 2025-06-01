#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script modifies the tc-apply.yml daemonset definition, applies it, then
# waits for it to complete so completion time can be documented.
# The durration of the test is defined by the sleep within the 2nd of 3
# initContainers.

SLEEP=$(grep sleep tc-apply.yml | head -1 | awk '{print $3}')
echo "Test will run for $SLEEP seconds, ~$((SLEEP / 60)) minutes"

NEW_SLEEP=$((SLEEP + 1))
sed -ie "s/sleep ${SLEEP}/sleep ${NEW_SLEEP}/g" tc-apply.yml
kubectl apply -f tc-apply.yml

sleep 10
kubectl -n kube-system get pods | grep apply | grep Init | wc -l

echo "Test started at $(date +'%Y-%m-%dT%H:%M')"
sleep ${SLEEP}

# Wait for all the containers to get out of Init state
while (true); do
    kubectl -n kube-system get pods | grep apply | grep Init >>/dev/null
    if [[ $? -ne 0 ]]; then
        break
    fi
    sleep 20
done
echo "Test ended at $(date +'%Y-%m-%dT%H:%M')"

# Ensure the rsources are removed
kubectl delete -f tc-apply.yml
