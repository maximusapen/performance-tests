#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************


D_COUNT=0
while (true); do
    if [[ $D_COUNT -le 0 ]]; then
        C_COUNT=$(sudo crictl ps | wc -l)
        echo "`date +%Y%m%d-%H%M%S` containers $C_COUNT"
        time crictl inspect `sudo crictl ps -q -a` | wc -l
        D_COUNT=10
    else
        D_COUNT=$((D_COUNT-1))
    fi
    SECONDS=0
    LAST=0
    for i in `sudo crictl ps | grep -v "CONTAINER ID" | tee pleg.docker.lst | cut -d" " -f1`; do
        DUR1ST=$((SECONDS-LAST))
        sudo crictl inspect $i > /dev/null
        if [[ $? -ne 0 ]]; then
            echo "ERROR: PLEG test error on $i"
        fi
        DUR=$((SECONDS-LAST))
        if [[ $DUR -gt 1 ]]; then
            if [[ $LAST -eq 0 ]]; then
                echo "`date +%Y%m%d-%H%M%S` ERROR: $i $SECONDS $LAST $DUR $DUR1ST - FIRST"
            else
                echo "`date +%Y%m%d-%H%M%S` ERROR: $i $SECONDS $LAST $DUR"
            fi
            grep $i pleg.docker.lst
        fi
        LAST=$SECONDS
    done
    duration=$SECONDS
    echo "`date +%Y%m%d-%H%M%S` `printf %02d:%02d $((duration/60)) $((duration%60))`"
done
