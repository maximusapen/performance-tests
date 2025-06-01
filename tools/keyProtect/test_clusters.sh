#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019, 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Following KUBECONFIG on performance client.  Modify as appropriate.

source ./test_conf.sh

for i in $(seq ${secretStart} ${secretEnd}); do
    echo
    date
    echo "*** Processing secret ${i} ***"

    cd secret

    # Add one secret
    ./add_secrets_5.sh ${i} ${i}

    # Get all secrets for all clusters with get_secrets.sh, not get_secrets_6.sh which run in background threads.
    ./get_secrets_6.sh

    cd ..

    # Restart all masters
    cd config

    SECONDS=0
    # Pass in optional secret id for logging purpose
    ./restart_masters.sh ${i}
    duration=${SECONDS}
    echo "Restart time Secret ${i} : ${SECONDS} sec"
    echo "Restart time ${i} DEKs:  $(($duration / 60)) minutes and $(($duration % 60)) seconds "

    cd ..
done
