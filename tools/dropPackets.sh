#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Assumes tcp, but not udp, ports

if [[ $# < 1 || $# > 2 ]]; then
    echo "Usage: dropPackets.sh port[,port[,port]] [ip]"
    exit 1
fi

ports=""
if [[ $1 == *","* ]]; then
    ports=" -m multiport --dports $1"
else
    ports="--dport $1"
fi

ip=""
if [[ $# == 2 ]]; then
    ip="-d $2"
fi

# trap ctrl-c and call ctrl_c()
trap ctrl_c INT

function enableNet() {
    sudo iptables -D OUTPUT -p tcp $ports $ip -j DROP
    echo "Enabled at `date`"
}

function ctrl_c() {
    echo "** Trapped CTRL-C"
    enableNet
    exit
}

DELAY=30

if [[ $# -eq 1 ]]; then
    DELAY=$1
fi

echo "Disabling for $DELAY seconds at `date`"
sudo iptables -I OUTPUT 1 -p tcp $ports $ip -j DROP
#-m state --state NEW,ESTABLISHED -j DROP 
sleep $DELAY
enableNet
