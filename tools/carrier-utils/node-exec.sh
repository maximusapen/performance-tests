#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to execute commands on 1 or more carrier hosts
# Requires ~/.ssh/config to be setup so that no pararameters are needed for ssh,
# each host to have the proper ~/.ssh/authorized_keys for auto login, and
# a /etc/hosts file with entries for each host you are try to reach.
# Example ~/.ssh/config file
#    User <user id>
#    Host stage-dal*
#      UserKnownHostsFile /dev/null
#      StrictHostKeyChecking no
#      IdentityFile ~/.ssh/id_rsa
#      IdentitiesOnly yes
#      LogLevel FATAL
#      ServerAliveInterval 120

START_WORKER=1001
END_WORKER=8003
NO_NEWLINE=""
NUMBERS=""

if [[ $# -lt 2 ]]; then
    echo "USAGE: node-exec.sh <command> <stage carrier #> [-][all|worker{default}|cruiser|armada|master|haproxy|dal## [<start worker number> [<end worker number> [<any text says don't issue newline after host name>]]]]"
    echo "       '-' after <stage carrier #> drops CR after host name allowing for single line per host output"
    exit
fi

COMMAND="$1"
CARRIER="carrier$2"
shift
shift
if [[ $# -ge 1 && $1 == "-" ]]; then
    NO_NEWLINE="-n"
    shift
fi

# START code in common with other scripts like node-scp.sh
if [[ $CARRIER == "carrier1" ]]; then
    START_WORKER=37
    END_WORKER=111
fi

HOST_FILTER="worker"
if [[ $# -ge 1 ]]; then
    case $1 in

   "all")
        echo "doing all"
        HOST_FILTER=""
        ;;
    "worker")
        HOST_FILTER="worker"
        ;;
    "cruiser")
        HOST_FILTER="worker-1"
        if [[ $CARRIER == "carrier1" ]]; then
            echo "'cruiser' not supported for carrier1"
            exit
        fi
        ;;
    "armada")
        HOST_FILTER="worker-8"
        if [[ $CARRIER == "carrier1" ]]; then
            echo "'armada' not supported for carrier1"
            exit
        fi
        ;;
    "master")
        HOST_FILTER="master"
        ;;
    (dal[0-9][0-9])
        HOST_FILTER="$1-$CARRIER-worker"
        if [[ $# -ge 2 ]]; then
            START_WORKER=$2
            NUMBERS="true"
        fi
        if [[ $# -ge 3 ]]; then
            END_WORKER=$3
        fi
        ;;
    "haproxy")
        HOST_FILTER="haproxy"
        ;;
   (*)
       NUMBERS="true"
       START_WORKER=$1 
       if [[ $# -ge 2 ]]; then
           END_WORKER=$2
       fi
       ;;
   esac

   if [[ ${START_WORKER} -gt ${END_WORKER} ]]; then
       echo "Start worker (${START_WORKER}) must be greater then or equal to end worker (${END_WORKER})"
       exit 1
   fi
fi

for i in `egrep -a "stage-[a-z]*[0-9]*-${CARRIER}" /etc/hosts | grep -v prestage | grep "${HOST_FILTER}" | tr "\t" " " | sort -k 2 | awk '{print $2}'`; do
    if [[ -n $NUMBERS ]]; then
        WORKER=$(echo "${i}" | sed -e "s/^.*-//g")
        #echo "WORKER=${WORKER}"
        if [[ ${WORKER} -lt ${START_WORKER} || ${WORKER} -gt ${END_WORKER} ]]; then
            continue
        fi
    fi
    # END code in common with other scripts like node-scp.sh
    echo $NO_NEWLINE "$i "
    ssh $i "$COMMAND"
done
