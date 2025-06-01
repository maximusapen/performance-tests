#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2018, 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

#for i in `ls backup/*2020-11-19-10-41.log`; do echo $i; awk '/2020-11-18 21:59/{ print NR; exit }' ${i}; done
#tail -n +51750 backup/etcd-5node-77q5flk6k9.2020-11-19-10-41.log | egrep -v "purged file|etcdserver: start to snapshot| etcdserver: saved snapshot|etcdserver: compacted raft log at" | grep "etcdserver: failed to send out heartbeat on time" | cut -d: -f1-2 | uniq -c
# Convert time into seconds
#tail -n +36741 backup/etcd-5node-flsh5jf9lz.2020-11-19-10-41.log | egrep -v "purged file|etcdserver: start to snapshot| etcdserver: saved snapshot|etcdserver: compacted raft log at" | grep "took too long" | egrep -v "took too long \([12][0-9][0-9].[0-9]*ms" | sed -e "s/^\([0-9-]*\) \([0-9:]*\).*took too long (\(.*\)).*$/\1 \2 \3/g" | sed -e "s/ \([0-9]*.[0-9]\)[0-9]*s/ \1s/g" -e "s/ \([0-9]*\).[0-9]*ms/ 0.\1s/g" |less

capture_date=$1
start_time=$2
end_time=$3

ignored_errors="purged file|etcdserver: start to snapshot| etcdserver: saved snapshot|etcdserver: compacted raft log at"
get_errors="compactor:|rafthttp|took too long|etcdserver: failed to send out heartbeat on time|mvcc:|grpc:"

total_lines=0
rm -rf backup/${capture_date}.etcd.test.errors backup/${capture_date}.etcd.test.summary.txt
for log_file in `ls backup/*${capture_date}.log`; do 
    echo "Processing: ${log_file}"
    lines=$(sed -n "/${start_time}/,/${end_time}/p" ${log_file} | egrep -v "${ignored_errors}" | tee -a backup/${capture_date}.etcd.test.errors | wc -l | awk '{print $1}')
    total_lines=$((total_lines+lines))
    if [[ ${lines} -eq 0 ]]; then
        echo "  No lines collected from ${log_file}"
    fi
done

if [[ ${total_lines} -eq 0 ]]; then
    echo "ERROR: No lines found in etcd logs"
    exit 1
fi
#2020-11-24 22:25:48.202662 I | rafthttp: established a TCP streaming connection with peer 299e64b46bd13289 (stream MsgApp v2 writer)
#WARNING: 2020/11/24 22:25:49 grpc: Server.processUnaryRPC failed to write status: connection error: desc = "transport is closing"
#-e "s#^\([0-9]*\)/\([0-9]*\)/\([0-9]*\) #\1-\2-\3 #g" 

egrep "${get_errors}" backup/${capture_date}.etcd.test.errors | egrep -v "took too long \([12][0-9][0-9].[0-9]*ms" | sed  -e "s/^\([0-9-]*\) \([0-9]*:[0-9]*\).*mvcc:.*$/\1 \2 na mvcc/g" -e "s#WARNING: \([0-9]*\)/\([0-9]*\)/\([0-9]*\) \([0-9]*\):\([0-9]*\):.*#\1-\2-\3 \4:\5 na grpc#g" -e "s/^\([0-9-]*\) \([0-9]*:[0-9]*\).*compactor:.*$/\1 \2 na compactor/g" -e "s/^\([0-9-]*\) \([0-9]*:[0-9]*\).*took too long (\(.*\)).*$/\1 \2 \3 toolong/g" -e "s/^\([0-9-]*\) \([0-9]*:[0-9]*\).*heartbeat.*timeout for \(.*\), .*$/\1 \2 \3 heartbeatissue/g" | sed -e "s/ \([0-9]*.[0-9]\)[0-9]*s/ \1s/g" -e "s/ \([0-9]*\).[0-9]*ms/ 0.\1s/g" -e "s/^\([0-9-]*\) \([0-9]*:[0-9]*\).*rafthttp:.*$/\1 \2 na rafthttp/g" | cut -d" " -f1-2,4 | sort | uniq -c >> backup/${capture_date}.etcd.test.summary.txt

echo "Etcd error data: backup/${capture_date}.etcd.test.summary.txt"
