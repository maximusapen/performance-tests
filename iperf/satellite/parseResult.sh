#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# Parse all client results output in json and returns the min, max and total throughput
# Total throughput only make sense if the results file are from a run with parallel connections.
if [[ $# -lt 1 ]]; then
    echo
    echo "Usage: ./parseResult.sh <results_directory>"
    echo
    exit 1
fi

# Directory with iperfclient output file in json format from getResult.sh
resultsDir=$1

declare -i nPod=0
declare -i MbitPSec
declare -i MinMbitPSec=9999
declare -i MaxMbitPSec=0
for results in $(ls ${resultsDir}/${tstamp}*); do
    podBitsPerSecond=$(grep -v "^args " $results | jq '.end.sum_received.bits_per_second')
    podBitsPerSecond=${podBitsPerSecond%.*}
    MbitPSec=$((podBitsPerSecond / 1000000))
    TotalMbitPSec=$((TotalMbitPSec + MbitPSec))
    if [[ ${MbitPSec} -eq 0 ]]; then
        # Result with error will have 0 throughput.  Skip.
        continue
    fi
    if [[ ${MbitPSec} -lt ${MinMbitPSec} ]]; then
        MinMbitPSec=${MbitPSec}
    fi
    if [[ ${MbitPSec} -gt ${MaxMbitPSec} ]]; then
        MaxMbitPSec=${MbitPSec}
    fi
    echo "Bandwidth for pod ${results#results/${tstamp}_}: $MbitPSec Mbits/second"
    ((nPod++))
done
declare -i AvgMbitPSec=$TotalMbitPSec/$nPod
echo
echo "Total throughput result is only valid for connections running in parallel for the period"
echo
echo "Min throughput / pod: $MinMbitPSec Mbits/second"
echo "Max throughput / pod: $MaxMbitPSec Mbits/second"
echo "Average throughput / pod: $AvgMbitPSec Mbits/second"
echo "Total throughput for $nPod pods: $TotalMbitPSec Mbits/second"
echo

# Check for error
errors=$(grep error ${resultsDir}/*)

if [[ ${errors} == "" ]]; then
    echo "No errors found"
else
    echo "errors:"
    grep error ${resultsDir}/*
fi
