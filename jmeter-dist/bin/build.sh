#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright Maximus Apen, 2025 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# This script will build the docker images and publish them to the stage repository

die() {
	printf '%s\n' "$1" >&2
	exit 1
}

environment="armada_performance"

# Allow the user to override the default duration/threads. (Pod default is specified at deploy time)
while :; do
	case $1 in
	-e | --environment) # Takes an option argument; ensure it has been specified.
		if [ "$2" ]; then
			environment=${environment}_$2
			shift
		else
			die 'ERROR: "--environment" requires a non-empty argument.'
		fi
		;;
	*) # Default case: No more options, so break out of the loop.
		break ;;
	esac

	shift
done

cd /performance/armada-perf

# Build master, slave and standalone images
docker build -t jmeter-dist-base -f jmeter-dist/imageCreate/base/Dockerfile .
docker build -t jmeter-dist-master -f jmeter-dist/imageCreate/master/Dockerfile .
docker build -t jmeter-dist-slave -f jmeter-dist/imageCreate/slave/Dockerfile .
docker build -t jmeter-dist-standalone -f jmeter-dist/imageCreate/standalone/Dockerfile .

docker tag jmeter-dist-master stg.icr.io/${environment}/jmeter-dist-master
docker tag jmeter-dist-slave stg.icr.io/${environment}/jmeter-dist-slave
docker tag jmeter-dist-standalone stg.icr.io/${environment}/jmeter-dist-standalone

# Push to stage registry
docker push stg.icr.io/${environment}/jmeter-dist-master
docker push stg.icr.io/${environment}/jmeter-dist-slave
docker push stg.icr.io/${environment}/jmeter-dist-standalone

cd -
