#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# Script to delete configmaps and secrets that match a prefix
#
# Usage: deleteConfigMaps.sh <Prefix to delete>
# All ConfigMaps and Serets starting with <prefix> will be deleted
#

###########################################################################
# main
###########################################################################
if [[ $# -ne 1 ]]; then
    echo "Usage: `basename $0` <prefix>"
    echo "All ConfigMaps and Serets starting with <prefix> will be deleted"
    exit 1
fi
PREFIX=$1

IFS=$'\n'

# Delete any configmaps that match the prefix
for configmaps in $(kubectl get configmaps)
do
    cmName=$(echo $configmaps |cut -d$' ' -f1)
    if [[ $cmName == $PREFIX* ]]
    then
      echo "kubectl delete configmap $cmName"
      if ! [ "$TESTING" = "true" ]
      then
        kubectl delete configmap $cmName
      fi
    else
      echo "ConfigMap $cmName does not match prefix of $PREFIX - ignoring"
    fi
done

# Delete any secrets that match the prefix
for secrets in $(kubectl get secrets)
do
    secret=$(echo $secrets |cut -d$' ' -f1)
    if [[ $secret == $PREFIX* ]]
    then
      echo "kubectl delete secret $secret"
      if ! [ "$TESTING" = "true" ]
      then
        kubectl delete secret $secret
      fi
    else
      echo "Secret $secret does not match prefix of $PREFIX - ignoring"
    fi
done

echo "Getting configmaps at end of deletion:"
kubectl get configmaps
echo "Getting secrets at end of deletion:"
kubectl get secrets
