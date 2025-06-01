#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2021, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Extracts automation schedule data and outputs a json file that is formated
# to make it easy to convert into html. The output json has two sections:
# - tests => For generating a table of a list of tests, which shows the client the test runs on which days of the week (columns)
# - clients => For generating a table of a list of clients, which shows the time of a day a particular test will be run.

days='"Monday" "Tuesday" "Wednesday" "Thursday" "Friday" "Saturday" "Sunday"'

# Get a list of all the tests, enabled or not
for day in ${days}; do
    #cat automation/schedule.json | jq ".${day}[] | .schedule[0].tests[0].test"
    cat automation/schedule.json | jq ".${day}[] | .schedule[].tests[].test"
done | sort -u > /tmp/perf.automation.tests.txt

# Generate tests table data
first_test=true
echo "{\"tests\" : ["
for test in `cat /tmp/perf.automation.tests.txt`; do
    if [[ ${first_test} == false ]]; then
        echo ","
    fi
    first_day="true"
    echo "{\"test\" : ${test}",
    realm="${test%\"}"
    realm="${realm#\"}"
    realm="${realm%%-*}"
    owner=$(grep "^${realm} " automation/assignments | awk '{print $2}')
    echo "\"owner\" : \"${owner}\"",
    echo "\"days\" : ["
    for day in ${days}; do
        scheduleLen=$(cat automation/schedule.json | jq ".${day}[] | select(.schedule[].tests[].test == ${test}) | .schedule | length")
        if [[ -n ${scheduleLen} ]]; then
            for ((i=0; i<${scheduleLen}; i++)); do
                client=$(cat automation/schedule.json | jq ".${day}[] | select(.schedule[${i}].tests[0].test == ${test}) | { day: ${day}, client: .client, enable: .schedule[${i}].tests[0].enable }" | sed -e "s/stgiks-dal10-//g" -e "s/client-//g")
                if [[ -n ${client} ]]; then
                    if [[ ${first_day} == false ]]; then
                        echo ","
                    fi
                    echo "${client}"

                    # Only move past the first day if we have actually
                    # output something for the first time.
                    first_day=false
                fi
            done
        else
            if [[ ${first_day} == false ]]; then
                echo ","
            fi
            echo "{\"day\" : ${day}}"
            first_day=false
        fi
    done
    echo "]"
    echo "}"
    first_test=false
done
echo "],"

# Get a list of all the clients
for day in ${days}; do
    cat automation/schedule.json | jq ".${day}[] | .client"
done | sort -u > /tmp/perf.automation.clients.txt

# Generate client table data
first_client=true
echo "\"clients\" : ["
for client in `cat /tmp/perf.automation.clients.txt`; do
    clientIsActive=$(cat automation/client.json | jq ".${client}.active")
    if [[ ${first_client} == false ]]; then
        echo ","
    fi
    first_day="true"
    # stgiks-dal10-perf0-client-01
    noPrefix=${client//stgiks-dal10-/}
    shortName=${noPrefix//client-/}
    echo "{\"client\" : ${client}, \"shortName\" : ${shortName}, \"active\": ${clientIsActive}",
    echo "\"days\" : ["
    for day in ${days}; do
                    if [[ ${first_day} == false ]]; then
                        echo ","
                    fi
        scheduleLen=$(cat automation/schedule.json | jq ".${day}[] | select(.client == ${client}) | .schedule | length")
        if [[ -n ${scheduleLen} ]]; then
            echo "{\"day\" : ${day}, \"tests\": ["
            first_test=true
            for ((i=0; i<${scheduleLen}; i++)); do
                test=$(cat automation/schedule.json | jq ".${day}[] | select(.client == ${client}) |  { test: .schedule[${i}].tests[0].test, enable: .schedule[${i}].tests[0].enable, test_hour: .schedule[${i}].test_hour }" | sed -e "s/stgiks-dal10-//g" -e "s/client-//g")
                if [[ -n ${test} ]]; then
                    if [[ ${first_test} == false ]]; then
                        echo ","
                    fi
                    echo "${test}"

                    # Only move past the first test if we have actually
                    # output something for the first time.
                    first_test=false
                fi
            done
            echo "]}"
        else
            echo "{\"day\" : ${day}, \"tests\": []}"
        fi
        first_day=false
    done
    echo "]}"
    first_client=false
done
echo "]}"
