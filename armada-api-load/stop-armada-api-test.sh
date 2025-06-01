#!/bin/bash

# Script to stop Jmeter test before the test is scheduled to terminate.
testPid=$(ps -ef | grep test-armada-api.sh | grep bash | awk '{print $2}')
jmeterPluginPid=$(ps -ef | grep apache-jmeter | grep -v ApacheJMeter.jar | grep -v grep | awk '{print $2}')
jmeterPid=$(ps -ef | grep apache-jmeter | grep ApacheJMeter.jar | awk '{print $2}')

echo jmeterPluginPid: ${jmeterPluginPid}
echo jmeterPid: ${jmeterPid}
echo testPid: ${testPid}

if [[ ${jmeterPluginPid} != "" ]]; then
    kill -9 ${jmeterPluginPid}
fi
if [[ ${jmeterPid} != "" ]]; then
    kill -9 ${jmeterPid}
fi
if [[ ${testPid} != "" ]]; then
    kill -9 ${testPid}
fi

printf "\n%s - All jmeter processes should be terminated now\n" "$(date +%T)"
ps -ef | grep test-armada-api.sh
ps -ef | grep jmeter
echo
