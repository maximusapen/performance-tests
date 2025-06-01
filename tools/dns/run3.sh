#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# params is the test type in params directory with default coredns
params=$1

# reportDir is option to send test output to out/${reportDir} instead of out
# Example, testing client on/off node with coredns, you can specify
# oncoredns or offcoredns to send results to out/oncoredns and out/offcoredns

reportDir=$2

if [ $# -lt 1 ]; then
    echo "No paramaters passed in.  Testing with default."
    params="coredns"
    reportDir=""
fi

echo
echo "Testing with:"
echo "  params=${params}"
echo "  reportDir=${reportDir}"
echo "Test result in out/${reportDir}/${params}-*"

mkdir -p out/${reportDir}

./runTest.sh ${params} multiple-svc ${reportDir}
./runTest.sh ${params} pod-ip ${reportDir}
./runTest.sh ${params} service ${reportDir}
./runTest.sh ${params} external ${reportDir}
./runTest.sh ${params} nx-domain ${reportDir}
