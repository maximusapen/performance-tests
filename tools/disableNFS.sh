#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# trap ctrl-c and call ctrl_c()
trap ctrl_c INT

function enableNet() {
    #sudo iptables -D OUTPUT -p tcp --dport 2049 -j DROP
    #sudo iptables -D OUTPUT -p tcp --dport 111 -j DROP
    sudo iptables -D OUTPUT -p udp -m multiport --dports 10053,111,2049,32769,875,892 -j DROP 
    sudo iptables -D OUTPUT -p tcp -m multiport --dports 10053,111,2049,32803,875,892 -j DROP 
    echo "NFS enabled at `date`"
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

echo "Disabling NFS for $DELAY seconds at `date`"
#sudo iptables -I OUTPUT 1 -p tcp --dport 2049 -j DROP
#sudo iptables -I OUTPUT 2 -p tcp --dport 111 -j DROP
sudo iptables -I OUTPUT -p udp -m multiport --dports 10053,111,2049,32769,875,892 -j DROP 
sudo iptables -I OUTPUT -p tcp -m multiport --dports 10053,111,2049,32803,875,892 -j DROP 
#-m state --state NEW,ESTABLISHED -j DROP 
sleep $DELAY
enableNet
