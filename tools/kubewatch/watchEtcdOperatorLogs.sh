#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Save etcd pods logs for all clusters. Waits for namespace to be at least 10 minutes old.

while (true); do
	now=$(date +'%Y-%m-%dT%H:%M:%S')
	nowEpoch=$(date +%s)
	echo "${now}: Checking for 'master-*' namespaces older than 10 minutes"
	for ns in `kubectl get ns --no-headers -o custom-columns="name:metadata.name" | grep "^master-"`; do
		if [[ ! -d $ns ]]; then
			nsCreationTimestamp=$(kubectl get ns ${ns} -o jsonpath='{.metadata.creationTimestamp}')
			nsStartTimeEpoch=$(date -d ${nsCreationTimestamp} +%s)
			nsDeltaMinutes=$(((nowEpoch-nsStartTimeEpoch)/60))
			if [[ ${nsDeltaMinutes} -ge 10 ]]; then
				mkdir ${ns}
				for pod in `kubectl -n ${ns} get pods -o custom-columns="name:metadata.name" --no-headers | grep "^etcd-"`; do
					if [[ ${pod} = "etcd-operator"* ]]; then
						kubectl -n ${ns} logs ${pod} etcd-operator > ${ns}/${pod}.${nowEpoch}.log
					else
						kubectl -n ${ns} logs ${pod} etcd > ${ns}/${pod}.${nowEpoch}.log
					fi
				done
			fi
		fi
	done

	sleep 10m
done
