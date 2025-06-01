#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
die() {
	printf '%s\n' "$1" >&2
	exit 1
}

waitForLB() {
	lb_host=$1
	counter=1
	lb_ready=false
	# Wait for up to 90 min for load balancer to be active
	until [[ $counter -gt 18 ]]; do
		if [[ -z ${node} ]]; then
			node=$(dig @1.1.1.1 +short ${lb_host} | head -n1)
		fi

		if [[ -n ${node} ]]; then
			resp=$(nc -zv -w10 ${node} ${port} 2>&1)
			if [[ $? -eq 0 ]]; then
				printf "%s - Load Balancer is available.\n" "$(date +%T)"
				lb_ready=true
				break
			fi
		fi

		printf "%s - %d. Waiting for Master Controller Load Balancer to become available.\n" "$(date +%T)" "${counter}"
		sleep 300
		((counter++))
	done

	if [ "${lb_ready}" != true ]; then
		printf "%s - Load Balancer is unavailable.\n" "$(date +%T)"
		return 1
	fi
}

# KUBECONFIG needs to be set
if [[ -z ${KUBECONFIG} ]]; then
	die 'ERROR: KUBECONFIG must be set.'
fi

setNamespace=""
namespace=""
charts="false"

rate=""
threads=""
duration=""
token=""

# Allow the user to override the default duration/threads. (Pod default is specified at deploy time)
while :; do
	case $1 in
	-c | --charts)
		charts="true"
		;;
	-d | --duration) # Takes an option argument; ensure it has been specified.
		if [ "$2" ]; then
			duration=$2
			shift
		else
			die 'ERROR: "--duration" requires a non-empty argument.'
		fi
		;;
	-r | --rate) # Takes an option argument; ensure it has been specified.
		if [ "$2" ]; then
			rate=$2
			shift
		else
			die 'ERROR: "--rate" requires a non-empty argument.'
		fi
		;;
	-t | --threads) # Takes an option argument; ensure it has been specified.
		if [ "$2" ]; then
			threads=$2
			shift
		else
			die 'ERROR: "--threads" requires a non-empty argument.'
		fi
		;;
	-n | --namespace) # Takes an option argument; ensure it has been specified.
		if [ "$2" ]; then
			setNamespace="-n $2"
			namespace="$2"
			shift
		else
			die 'ERROR: "--namespace" requires a non-empty argument.'
		fi
		;;
	*) # Default case: No more options, so break out of the loop.
		break ;;
	esac

	shift
done

# Communication with master-controller pod is via a loadbalancer; get the details
# For classic clusters we have an ip for the lb, for vpc/vpc-classic we have a hostname
node=$(kubectl get service ${setNamespace} jmeter-master-controller-svc -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
if [[ -z $node ]]; then
	printf "%s - Using load balancer hostname to communicate with controller pod\n" "$(date +%T)"
	hnm=$(kubectl get service ${setNamespace} jmeter-master-controller-svc -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
else
	printf "%s - Using load balancer ip to communicate with controller pod\n" "$(date +%T)"
fi

port=$(kubectl get service ${setNamespace} jmeter-master-controller-svc -o jsonpath='{.spec.ports[0].port}')
echo "Node: $node"
echo "Port: $port"

if [[ -z ${port} ]]; then
	die 'ERROR: Invalid port number.'
fi

# Wait for the load balancer to become avilable.
# Classic clusters should hopefully sail through this. vpc/vpc-classic will take a while
waitForLB ${hnm}

if [[ $? -ne 0 ]]; then
	die "ERROR: Port not accessible"
fi

mode=$(kubectl ${setNamespace} get pods | awk '{print $1}' | grep jmeter | grep -m 1 master | cut -d"-" -f2)

if [[ $mode == "dist" ]]; then
	mode=slave
elif [[ $mode != "standalone" ]]; then
	die "ERROR: Invalid mode discovered: ${mode}"
fi
echo "Mode: ${mode}"

# Get the pod default threads, rate and test duration if not specified by the user
if [[ -z ${rate} ]]; then
	rate=$(kubectl get cm ${setNamespace} -o jsonpath='{.items[0].data.JMETER_ARGS}' | sed -nr 's/.*TPUTMINS=([0-9]*).*/\1/p')
	if [[ -n ${rate} ]]; then
		rate=$((rate / 60)) # Temporarily convert to requests/s to match expected user input
	fi
fi

if [[ -z ${threads} ]]; then
	threads=$(kubectl get cm ${setNamespace} -o jsonpath='{.items[0].data.JMETER_ARGS}' | sed -nr 's/.*THREAD=([0-9]*).*/\1/p')
fi

if [[ -z ${duration} ]]; then
	duration=$(kubectl get cm ${setNamespace} -o jsonpath='{.items[0].data.JMETER_ARGS}' | sed -nr 's/.*RUNLENSEC=([0-9]*).*/\1/p')
fi

if [[ -z ${token} ]]; then
	token=$(kubectl get cm ${setNamespace} -o jsonpath='{.items[0].data.JMETER_ARGS}' | sed -nr 's/.*TOKEN=([a-zA-Z0-9]*).*/\1/p')
fi

master=""
# Trigger start of JMeter test.
printf "%s - Starting test with %s threads per slave.\n" "$(date +%T)" "${threads}"
echo -en "START ${threads} ${rate} ${duration} ${mode} ${namespace} ${token}" | nc -q 1 "${node}" "${port}"

# Results are not going to be ready for at least the test duration, so we might as well sleep for a bit
if [[ -n ${duration} ]]; then
	waitTime=$((${duration} + 120))
	printf "%s - Waiting for %ss.\n" "$(date +%T)" "${waitTime}"
	sleep ${waitTime}
fi

counter=1
results=""
# Now wait for up to 60 minutes for the JMeter test to complete and then receive the results
# via the master-controller nodeport connection
until [[ $counter -gt 180 ]]; do
	# Receive results from master-controller pod
	results=$(echo -en "RESULTS" | nc -q 1 "${node}" "${port}")

	if [[ -n ${results} ]]; then
		break
	fi
	printf "%s - %d. Waiting for results to become available.\n" "$(date +%T)" "${counter}"
	sleep 20
	((counter++))
done

printf "%s - Test complete.\n" "$(date +%T)"

if [[ -n ${results} ]]; then
	echo "${results}"

	if [[ $charts == "true" ]]; then
		echo "Generating charts"

		if [[ -z $master ]]; then
			master=$(kubectl get pods ${setNamespace} | grep jmeter-standalone-master- | awk '{print $1}')
		fi

		chartFile="charts.$(date "+%Y-%m-%d_%H%M%S").tar.gz"
		kubectl -n ${namespace} exec ${master} -- rm -rf charts
		kubectl -n ${namespace} exec ${master} -- jmeter -Jjmeter.reportgenerator.overall_granularity=1000 -g /jmeter/results/samples.out -o charts
		kubectl -n ${namespace} exec ${master} -- tar czf - charts >${chartFile}
		echo "Charts are at ${chartFile}"
	fi
else
	printf "%s - FAILURE: No results received!\n" "$(date +%T)"
	exit 1
fi
