#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Pre-reqs:
# - iperf3 installed on all hosts (sudo apt-get install iperf3)
# - Passwordless ssh access to all hosts (for these hosts they have armada user on
#   there so used ~/.ssh/config to point at armada key)
# - On the machine where running the test you need jq 1.5 installed - I had issues with 1.3 (which was on my vagrant):
#    wget https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64
#    chmod +x jq-linux64
#    sudo mv jq-linux64 $(which jq)
#
# Servers being tested are as follows:
# dev-sng01-perf1-client-01 10.66.160.42 119.81.63.99 vsi
# dev-sng01-perf2-client-01 10.66.160.45 119.81.63.102 bm
# dev-syd04-perf1-client-01 10.63.24.2 130.198.65.205 vsi
# dev-syd04-perf2-client-01 10.63.24.25 130.198.65.203 bm

SNG_VSI_PRIVATE=10.66.160.42
SNG_VSI_PUBLIC=119.81.63.99

SYD_VSI_PRIVATE=10.63.24.2
SYD_VSI_PUBLIC=130.198.65.205

SNG_BM_PRIVATE=10.66.160.45
SNG_BM_PUBLIC=119.81.63.102

SYD_BM_PRIVATE=10.63.24.25
SYD_BM_PUBLIC=130.198.65.203


SNG_VSI_HOST=dev-sng01-perf1-client-01
SYD_VSI_HOST=dev-syd04-perf1-client-01
SNG_BM_HOST=dev-sng01-perf2-client-01
SYD_BM_HOST=dev-syd04-perf2-client-01

RESULTS_FILE=NetworkBandwidthResults.csv
PING_RESULTS_FILE=NetworkPingResults.csv

function executeIPerf3 {
  DATE=$(date --utc +%FT%TZ)
  RESULT=$(ssh armada@$CLIENT_HOST "iperf3 -c $SERVER_IP -P $THREADS -J")
  BANDWIDTH=$(echo ${RESULT} | jq '.end.sum_sent.bits_per_second')
  RETRANSMITS=$(echo ${RESULT} | jq '.end.sum_sent.retransmits')
  CPU_CLIENT=$(echo ${RESULT} | jq '.end.cpu_utilization_percent.host_total')
  CPU_SERVER=$(echo ${RESULT} | jq '.end.cpu_utilization_percent.remote_total')
  echo "$DATE,$MACHINE_TYPE,$LOCAL_LOCATION,$CLIENT_HOST,$REMOTE_LOCATION,$SERVER_IP,$NETWORK,$THREADS,$BANDWIDTH,$RETRANSMITS,$CPU_CLIENT,$CPU_SERVER" >> $RESULTS_FILE
}

function executePing {
  DATE=$(date --utc +%FT%TZ)
  OUTPUT=$(ssh armada@$CLIENT_HOST "sudo ping -f -c 1000 -s 1500 -l 4 $SERVER_IP")

  # This cut is pretty brittle, but it works and it doesn't look like ping gives output in better format
  RESULTS=$(echo ${OUTPUT} | cut -d$'=' -f2 | cut -d$' ' -f2)
  MIN=$(echo $RESULTS | cut -d$'/' -f1)
  AVG=$(echo $RESULTS | cut -d$'/' -f2)
  MAX=$(echo $RESULTS | cut -d$'/' -f3)
  MDEV=$(echo $RESULTS | cut -d$'/' -f4)
  echo "$DATE,$MACHINE_TYPE,$LOCAL_LOCATION,$CLIENT_HOST,$REMOTE_LOCATION,$SERVER_IP,$NETWORK,$MIN,$AVG,$MAX,$MDEV" >> $PING_RESULTS_FILE
}

# First kill iperf3 and restart on all hosts
echo 'Restarting iperf3 on '$SNG_VSI_HOST','$SYD_VSI_HOST','$SNG_BM_HOST',' $SYD_BM_HOST
ssh armada@$SNG_VSI_HOST 'pkill iperf3; sleep 2; nohup iperf3 -sV > /home/armada/iperf3_server.txt 2>&1 &'
ssh armada@$SYD_VSI_HOST 'pkill iperf3; sleep 2; nohup iperf3 -sV > /home/armada/iperf3_server.txt 2>&1 &'
ssh armada@$SNG_BM_HOST 'pkill iperf3; sleep 2; nohup iperf3 -sV > /home/armada/iperf3_server.txt 2>&1 &'
ssh armada@$SYD_BM_HOST 'pkill iperf3; sleep 2; nohup iperf3 -sV > /home/armada/iperf3_server.txt 2>&1 &'
echo 'iperf3 restarted on '$SNG_VSI_HOST','$SYD_VSI_HOST','$SNG_BM_HOST',' $SYD_BM_HOST


echo 'Date,Machine_Type,Client_Location,Client_Host,Server_Location,Server_IP,Network,Threads,Bandwidth(bits_per_second),Retransmits,CPU_Client,CPU_Server' >> $RESULTS_FILE
echo 'Date,Machine_Type,Client_Location,Client_Host,Server_Location,Server_IP,Network,Min(ms),Mean(ms),Max(ms),MDev' >> $PING_RESULTS_FILE

# VSI Tests SNG->SYD
echo 'Running SNG->SYD PRIVATE VSI'
MACHINE_TYPE='VSI'
LOCAL_LOCATION='SNG'
REMOTE_LOCATION='SYD'
CLIENT_HOST=$SNG_VSI_HOST
SERVER_IP=$SYD_VSI_PRIVATE
NETWORK='PRIVATE'

executePing

THREADS='1'
executeIPerf3
THREADS='10'
executeIPerf3
THREADS='20'
executeIPerf3

echo 'Running SNG->SYD PUBLIC VSI'

SERVER_IP=$SYD_VSI_PUBLIC
NETWORK='PUBLIC'

executePing

THREADS='1'
executeIPerf3
THREADS='10'
executeIPerf3
THREADS='20'
executeIPerf3

# VSI Tests SYD->SNG
echo 'Running SYD->SNG PRIVATE VSI'
LOCAL_LOCATION='SYD'
REMOTE_LOCATION='SNG'
CLIENT_HOST=$SYD_VSI_HOST
SERVER_IP=$SNG_VSI_PRIVATE
NETWORK='PRIVATE'

executePing

THREADS='1'
executeIPerf3
THREADS='10'
executeIPerf3
THREADS='20'
executeIPerf3


echo 'Running SYD->SNG PUBLIC VSI'
SERVER_IP=$SNG_VSI_PUBLIC
NETWORK='PUBLIC'

executePing

THREADS='1'
executeIPerf3
THREADS='10'
executeIPerf3
THREADS='20'
executeIPerf3

# BM Tests SNG-> SYD
echo 'Running SNG->SYD PRIVATE BM'
MACHINE_TYPE='BM'
LOCAL_LOCATION='SNG'
REMOTE_LOCATION='SYD'
CLIENT_HOST=$SNG_BM_HOST
SERVER_IP=$SYD_BM_PRIVATE
NETWORK='PRIVATE'

executePing

THREADS='1'
executeIPerf3
THREADS='10'
executeIPerf3
THREADS='20'
executeIPerf3

echo 'Running SNG->SYD PUBLIC BM'

SERVER_IP=$SYD_BM_PUBLIC
NETWORK='PUBLIC'

executePing

THREADS='1'
executeIPerf3
THREADS='10'
executeIPerf3
THREADS='20'
executeIPerf3

# BM Tests SYD->SNG
echo 'Running SYD->SNG PRIVATE BM'
LOCAL_LOCATION='SYD'
REMOTE_LOCATION='SNG'
CLIENT_HOST=$SYD_BM_HOST
SERVER_IP=$SNG_BM_PRIVATE
NETWORK='PRIVATE'

executePing

THREADS='1'
executeIPerf3
THREADS='10'
executeIPerf3
THREADS='20'
executeIPerf3


echo 'Running SYD->SNG PUBLIC BM'
SERVER_IP=$SNG_BM_PUBLIC
NETWORK='PUBLIC'

executePing

THREADS='1'
executeIPerf3
THREADS='10'
executeIPerf3
THREADS='20'
executeIPerf3
