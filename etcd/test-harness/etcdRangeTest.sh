#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2017, 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

. etcd-perftest-config

dir=`dirname $0`

#if [[ $# -ne 2 ]]; then
#    echo "Usage: `basename $0` <endpoints> <comment>"
#    exit 1
#fi

ETCDCTRL=/usr/local/bin/etcdctl

ENDPOINTS=169.61.187.36:30654
ENDPOINTS=${ETCD_ENDPOINTS}
COMMENT="Range"
KEYSIZE=200
VALSIZE=500
GETTOTAL=100000
PUTTOTAL=100000
PUTRATE=0
GETRATE=0

ETCDCTL_API=3
TESTS="--consistency l,--consistency s"
OIFS=$IFS
IFS=','
TESTARRAY=(${TESTS})
IFS=$OIFS

BASEKEY="/perf/rangetest/"
# TODO generate key based on $VALSIZE
VALUE="0********1*********2"

BASELEN=${#BASEKEY}
FILLLEN=$((KEYSIZE-BASELEN))
FILLFMT="%0${FILLLEN}d"
SEQFILLFMT="%0${FILLLEN}g"
PATTERN="${BASEKEY}%${FILLFMT}[$GETTOTAL];[a-z0-9]{20}"
CLIENTS=1000
CLIENTS=100

if [[ $1 == "setup" ]]; then
    echo "Load database"
    etcd-driver pattern $ETCDCREDS --endpoints $ENDPOINTS --total $GETTOTAL --pattern $PATTERN --put-rate $PUTRATE
else 
    #./defrag-etcd.sh

    dt=$(date +"%Y-%m-%d-%H-%M")
    echo "Start: ${dt} range tests with ${CLIENTS} clients"
    for j in "${!TESTARRAY[@]}"; do
        for (( i=1; i<=$GETTOTAL; i=i*10 )); do
            SPACE=$i
            lastKey=$(seq -f ${BASEKEY}${SEQFILLFMT} $i $i)
            echo "Range get $SPACE sequential keys $GETTOTAL requests in key space of $GETTOTAL with ${TESTARRAY[j]} at `date "+%Y%d%m-%H%M.%S"`, $COMMENT"
            etcd-benchmark range $BASEKEY $lastKey  $ETCDCREDS --endpoints $ENDPOINTS --csv-file etcdRangeResults.csv --file-comment "${COMMENT}" --total $GETTOTAL --conns=100 --clients=${CLIENTS} ${TESTARRAY[j]}
            # This code adds the number of keys to be read to the end of the csv file since that is a key piece of data that doesn't come from etcd-benchmark
            lastLine=$(tail -1 etcdRangeResults.csv)
            sed "s/$lastLine/$lastLine,$SPACE/" etcdRangeResults.csv
        done
    done

    #$ETCDCTRL $ETCDCREDS --endpoints $ENDPOINTS del $BASEKEY $(seq -f ${BASEKEY}${SEQFILLFMT} $GETTOTAL $GETTOTAL)
fi
echo "End: $(date +'%Y-%m-%d-%H-%M')"
