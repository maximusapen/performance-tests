#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Watch for long running kubx-deployer pods

while (true); do
	now=$(date +'%Y-%m-%dT%H:%M:%S')
	nowEpoch=$(date +%s)
	echo "${now}: Checking long running kubx-deployer deploy pods"
	for i in `kubectl -n kubx-deployer get pod -l operation=deploy -o custom-columns="name:metadata.name" --no-headers`; do
		podState=$(kubectl -n kubx-deployer get pod $i -o jsonpath='{.status.phase}')
		if [[ ${podState} == "Running" ]]; then
			startTime=$(kubectl -n kubx-deployer get pod $i -o jsonpath='{.status.startTime}')
			startTimeEpoch=$(date -d ${startTime} +%s)
			deltaSeconds=$((nowEpoch-$startTimeEpoch))
			deltaMinutes=$((deltaSeconds/60))
			deltaHours=$((deltaMinutes/60))
			if [[ ${deltaHours} -ge 2 ]]; then
				kubectl -n kubx-deployer logs $i kubx-deployer > $i.${nowEpoch}.log
				clusterId=$(grep "Starting Deployer Run - Cluster: " $i.${nowEpoch}.log| sed -e "s/^.*Cluster: \([0-9a-z]*\), .*$/\1/g")
				if [[ ${#clusterId} -gt 0 && (  ! -f ${clusterId} || ${deltaHours} -gt 20 ) ]]; then
					echo "${now}: $i $startTime $startTimeEpach $deltaMinutes $deltaHours ${clusterId}"
					mv $i.${nowEpoch}.log ${clusterId}.$i.${nowEpoch}.log
		  			kubectl -n kubx-deployer get pod $i -o yaml > ${clusterId}.$i.${nowEpoch}.yml
		  			kubectl -n kubx-deployer get pods > ${clusterId}.pods.${nowEpoch}.yml
					for pod in `kubectl -n master-${clusterId} get pods -o custom-columns="name:metadata.name" --no-headers | grep "^etcd-"`; do
						if [[ ${pod} = "etcd-operator"* ]]; then
							kubectl -n master-${clusterId} logs ${pod} etcd-operator > ${clusterId}.${clusterId}.${pod}.${nowEpoch}.log
						else
							kubectl -n master-${clusterId} logs ${pod} etcd > ${clusterId}.${clusterId}.${pod}.${nowEpoch}.log
						fi
					done
					for dep in `kubectl -n master-${clusterId} get deployments -o custom-columns="name:metadata.name" --no-headers`; do
						kubectl -n master-${clusterId} get deployments ${dep} >  ${clusterId}.${clusterId}.${dep}.${nowEpoch}.deployment.log
					done
					touch ${clusterId}
				else
					echo "${now}: $i $startTime $startTimeEpach $deltaMinutes $deltaHours"
					rm $i.${nowEpoch}.log
				fi
			fi
		fi
	done

	sleep 1h
done

#Mac: date -j -f '%Y-%m-%dT%H:%M:%S' '2020-03-02T16:07:45' +%s
#Linux: date -d 2020-03-02T16:07:45 +%s
