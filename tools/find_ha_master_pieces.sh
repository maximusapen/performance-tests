#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Dumps out a list of the pieces associated with cruisers that would match the filter
# If '-' is specified it prints the cruiser name associated with each piece

USAGE="Usage: find_ha_master_pieces.sh <cruiser name prefix> [-]"

NAME_ONLY=
if [[ $# -ge 1 ]]; then
    CRUISER_NAME_PREFIX=$1

    if [[ $# -eq 2 ]]; then
         NAME_ONLY="true"
    fi
else
    echo "$USAGE"
    exit 1
fi

function filter_results {
    if [[ -z ${NAME_ONLY} ]]; then
        cat
    else
        if [[ $# -eq 2 ]]; then
            tr -s " " | cut -d" " -f$1 | sed -e "s/^.*${CRUISER_NAME_PREFIX}/${CRUISER_NAME_PREFIX}/g" | eval $2
        else
            tr -s " " | cut -d" " -f$1 | sed -e "s/^.*${CRUISER_NAME_PREFIX}/${CRUISER_NAME_PREFIX}/g"
        fi
    fi
}

function print_type {
    if [[ -z ${NAME_ONLY} ]]; then
        echo "$1"
    fi
}

cnt=0
# Find master and openvpnserver pieces
print_type "# kubx-masters deployments"
kubectl -n kubx-masters get deployments | grep "${CRUISER_NAME_PREFIX}" | filter_results 1

print_type "# kubx-masters replicasets"
kubectl -n kubx-masters get rs | grep "${CRUISER_NAME_PREFIX}" | filter_results 1 'rev | cut -d"-" -f2-99 | rev'

print_type "# kubx-masters pods"
kubectl -n kubx-masters get pods -o wide | grep "${CRUISER_NAME_PREFIX}" | filter_results 1 'rev | cut -d"-" -f3-99 | rev'

print_type "# kubx-masters services"
kubectl -n kubx-masters get svc | grep "${CRUISER_NAME_PREFIX}" | filter_results 1

print_type "# kubx-masters configmaps"
kubectl -n kubx-masters get cm | grep "${CRUISER_NAME_PREFIX}" | filter_results 1 'sed -e "s/-[certsconfigvpnclusterinfo-]*$//g"'

print_type "# kubx-cit configmaps"
kubectl -n kubx-cit get cm | grep "${CRUISER_NAME_PREFIX}" | filter_results 1 'rev | cut -d"-" -f2-99 | rev'

# Find etcdclusters pieces
print_type "# kubx-etcd- etcdclusters"
kubectl get etcdclusters --all-namespaces | grep kubx-etcd- | grep "${CRUISER_NAME_PREFIX}" | filter_results 2

print_type "# kubx-etcd- pods"
kubectl get pods --all-namespaces -o wide | grep kubx-etcd- | grep "${CRUISER_NAME_PREFIX}" | filter_results 2 'rev | cut -d"-" -f2-99 | rev'

print_type "# kubx-etcd- services"
kubectl get svc --all-namespaces | grep kubx-etcd- | grep "${CRUISER_NAME_PREFIX}" | filter_results 2 'sed -e "s/-client-service-np$//g" -e "s/-client$//g"'

print_type "# kubx-etcd- secrets"
kubectl get secrets --all-namespaces | grep kubx-etcd- | grep "${CRUISER_NAME_PREFIX}" | filter_results 2 'sed -e "s/-[a-z]*-tls//g"'

print_type "# kubx-etcd- etcdbackups"
kubectl get etcdbackups --all-namespaces | grep kubx-etcd- | grep "${CRUISER_NAME_PREFIX}" | filter_results 2

print_type "# kubx-etcd- etcdrestores"
kubectl get etcdrestores --all-namespaces | grep kubx-etcd- | grep "${CRUISER_NAME_PREFIX}" | filter_results 2

print_type "# kubx-etcd- cronjobs"
kubectl get cronjobs  --all-namespaces | grep kubx-etcd- | grep "${CRUISER_NAME_PREFIX}" | filter_results 2

if [[ -z ${NAME_ONLY} ]]; then
    print_type "# kubx-etcd- jobs"
    kubectl get jobs  --all-namespaces | grep kubx-etcd- | grep "${CRUISER_NAME_PREFIX}"

    # Assumes .aws defined with credendials
    # TODO make carrier independent
    if [[ -f ~/.aws/credentials ]]; then
        print_type "# COS backups"
        ETCD_BACKUPS_ENDPOINT=https://s3-api.us-geo.objectstorage.softlayer.net
        ETCD_BACKUP_BUCKET=stage-prod09-carrier1-etcdbackups
        sudo /usr/local/bin/aws --endpoint "$ETCD_BACKUPS_ENDPOINT" s3 ls --recursive s3://$ETCD_BACKUP_BUCKET | grep ${CRUISER_NAME_PREFIX}
    fi
fi

# Ensure that last kubectl command failure doesn't cause this script to return failure
exit 0
