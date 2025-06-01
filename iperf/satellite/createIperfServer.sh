#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# Create specified number of iperfServer

if [[ $# -lt 2 ]]; then
    echo
    echo "Usage: ./createIperfServer [ dal09 | syd05 ] <number_of_servers> "
    echo
    exit 1
fi

source envFile
dataCentre=$1

dal09_public_vlan=2951366
syd05_public_vlan=3027946
che01_public_vlan=3076070

if [[ ${dataCentre} == "dal09" ]]; then
    vlan=${dal09_public_vlan}
elif [[ ${dataCentre} == "syd05" ]]; then
    vlan=${syd05_public_vlan}
elif [[ ${dataCentre} == "che01" ]]; then
    vlan=${che01_public_vlan}
    
else
    echo "DataCentre ${dataCentre} not supported"
    exit 1
fi

# Number of servers running in parallel
concurrency=$2

if [[ -z ${SERVER_KUBECONFIG} ]]; then
    echo "You need to export SERVER_KUBECONFIG"
    exit 1
fi

echo Using SERVER_KUBECONFIG ${SERVER_KUBECONFIG}
export KUBECONFIG=${SERVER_KUBECONFIG}

cd ${perf_dir}/iperf/bin

startpod=1
for i in $(seq ${startpod} ${concurrency}); do
    echo iperfserver.sh --pods 1 --id ${i} --vlan ${vlan}
    # Need to run this in sequence.  Otherwise may hit error:
    #     Using vlan 3027946 from --vlan 3027946
    #     Error: cannot re-use a name that is still in use
    ./iperfserver.sh --pods 1 --id ${i} --vlan ${vlan}
done
