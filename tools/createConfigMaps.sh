#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# Script to create lots of configmaps and secrets based on the contents of a directory
#
# Usage: createConfigMaps.sh <directory> <Number of sets to create> <Prefix to attach to names> [<etcd_endpoint>]
# Include <etcd_endpoint> if you want to count the keys in etcd during processing"
#

###########################################################################
# Create a kubernetes ConfigMap
# $1 = the name of the ConfigMap
# $2 = the file to create the ConfigMap from
function createConfigMapFromFile {
  NAME=$(kubifyName $1)
  echo "kubectl create configmap $NAME --from-file=$2"
  if ! [ "$TESTING" = "true" ]
  then
    kubectl create configmap $NAME --from-file=$2
  fi
}

###########################################################################
# Create a kubernetes secret
# $1 = the name of the secret
# $2 = the file to create the secret from
function createSecret {
  NAME=$(kubifyName $1)
  echo "kubectl create secret generic $NAME --from-file=$1=$2"
  if ! [ "$TESTING" = "true" ]
  then
    kubectl create secret generic $NAME --from-file=$1=$2
  fi
}

###########################################################################
# Function to remove underscores and upper case characters from a name
# $1 = The name to modify
function kubifyName {
  LOWER_NAME=${1,,}
  NO_UNDERS_NAME=$(echo $LOWER_NAME | tr _ -)
  echo $NO_UNDERS_NAME
}

###########################################################################
# Function to navigate a directory structure and populate an array with th filenames
# $1 = directory
function processDir {
  if ! [[ -d "$1" ]]
  then
    echo "processDir arg must be a directory"
    exit 1
  fi
  for file in $1/*
  do
    if [[ -d "${file}" ]]
    then
      processDir ${file}
    elif [[ -f "${file}" ]]
    then
      echo "Found file ${file}"
      FILES+=("${file}")
    fi
  done

}
###########################################################################
# Gets etcd stats using metrics endpoint
# $1 = Etcd endpoint
function countEtcd {
  statsFile=etcdStats.txt
  CURL_COMMAND="curl -L --cert /etc/kubernetes/cert/etcd.pem --key /etc/kubernetes/cert/etcd-key.pem --cacert /etc/kubernetes/cert/ca.pem $1/metrics > $statsFile"
  if [ "$TESTING" = "true" ]
  then
    echo $CURL_COMMAND
    return
  fi

  eval $CURL_COMMAND

  etcd_count=$(cat $statsFile |grep ^etcd_debugging_mvcc_keys_total | cut -d$' ' -f2)
  etcd_db_size=$(cat $statsFile | grep ^etcd_debugging_mvcc_db_total_size_in_bytes | cut -d$' ' -f2)
  etcd_db_size=$(printf '%.0f' $etcd_db_size)
  echo 'Count of etcd keys: ' $etcd_count
  export etcd_count

  echo 'etcd DB size: ' $etcd_db_size
  export etcd_db_size

  rm $statsFile
}

###########################################################################
# main
###########################################################################
if [[ $# -le 2 ]]; then
    echo "Usage: `basename $0` <directory> <Number of sets to create> <Prefix to attach to names> [<etcd_endpoint>]"
    echo "Include <etcd_endpoint> if you want to count the keys in etcd during processing"
    exit 1
fi

# Array that will be populated by processDir
FILES=()
processDir $1

# The number of sets of configMaps/secrets to create
SETS=$2
# Prefix to add to name of the configmaps/secrets
PREFIX=$3
ETCD_ENDPOINT=$4

STATS_FILE=CreateConfigMapStats.csv

if ! [ -z "$ETCD_ENDPOINT" ]
then
  countEtcd $ETCD_ENDPOINT
fi

TIME=$(date --utc +%FT%TZ)
echo "Starting test at $TIME"
SECONDS=0
kubectl get configmaps
GET_CM_DURATION=$SECONDS
SECONDS=0
kubectl get secrets
GET_SECRET_DURATION=$SECONDS
SECONDS=0
kubectl get pods --all-namespaces
GET_PODS_DURATION=$SECONDS
echo "Time, Duration(s), NumSets, KeyCount, DB Size(bytes), get cm Duration(s), get secret Duration(s), get Pods Duration (s)" >> $STATS_FILE
echo "$TIME,0,0,$etcd_count,$etcd_db_size,$GET_CM_DURATION,$GET_SECRET_DURATION,$GET_PODS_DURATION" >> $STATS_FILE

SECONDS=0
for (( i=1; i<=${SETS}; i=i+1 )); do
  for file in ${FILES[@]}
  do
    name=$(basename ${file})
    extension="${name##*.}"
    if [ "$extension" = "pem" ]
    then
     createSecret "test.$PREFIX.$i.secret.$name" ${file}
    else
      createConfigMapFromFile "test.$PREFIX.$i.cfmap.$name" ${file}
    fi
    done

    # Periodically print out the  stats
    if (( $i % 10 == 0 ))
    then
      duration=$SECONDS
      SECONDS=0
      kubectl get configmaps
      GET_CM_DURATION=$SECONDS
      SECONDS=0
      kubectl get secrets
      GET_SECRET_DURATION=$SECONDS
      SECONDS=0
      kubectl get pods --all-namespaces
      GET_PODS_DURATION=$SECONDS
      if ! [ -z "$ETCD_ENDPOINT" ]
      then
        countEtcd $ETCD_ENDPOINT
      fi

      TIME=$(date --utc +%FT%TZ)
      echo "Set $i complete"
      echo "$TIME,$duration,$i,$etcd_count,$etcd_db_size,$GET_CM_DURATION,$GET_SECRET_DURATION,$GET_PODS_DURATION" >> $STATS_FILE
      SECONDS=0
    fi
done

TIME=$(date --utc +%FT%TZ)
echo "Test completed at $TIME"
