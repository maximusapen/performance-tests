#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# Schedule tests based on run time, i.e. day of week and hour, from
# https://github.ibm.com/alchemy-containers/armada-performance-data/tree/master/automation/schedule.json.

if [ $# -eq 2 ]; then
    echo "This is a test with dayOfWeek and scheduleHour passed in"
    dayOfWeek=$1
    scheduleHour=$2
else
    echo "This is a real schedule.  Getting dayOfWeek and scheduleHour"
    dayOfWeek=$(date +%A)
    scheduleHour=$(date +%-H)
fi

echo "Checking test schedules for ${dayOfWeek} at ${scheduleHour} hour"

runTestSuite() {
    echo "Run testsuite"
    # Concurrent job for carriers with tests on only one client
    job="Schedule-Performance-TestSuite"
    # For carriers with more than one client, schedule to carrier queue job
    if [[ ${client} == *"perf3"* ]]; then
        job="${job}-Carrier3"
    elif [[ ${client} == *"perf4"* ]]; then
        job="${job}-Carrier4"
    fi
    jobURL="https://alchemy-testing-jenkins.swg-devops.com/view/Armada-performance/job/Armada-Performance/job/Automation/job/${job}"
    echo "Triggering ${job} job for ${TEST} with ${TEST_TEMPLATE} "
    curl -i -s -k -X POST --user armada.performance@uk.ibm.com:${STAGE_GLOBAL_ARMPERF_TEST_JENKINS_TOKEN} "${jobURL}/buildWithParameters?PERF_CLIENTS=${client}&TEST_TEMPLATE=${TEST_TEMPLATE}&TEST=${TEST}"
}

scheduleTests() {
    index=$1
    testSchedule=$(cat schedule.json | jq '.'${dayOfWeek}'['${index}']')
    if [[ ${testSchedule} == null ]]; then
        # No more test schedule
        return
    fi
    echo ${testSchedule}
    client=$(echo ${testSchedule} | jq .client | sed "s/\"//g")
    echo
    echo ${client}

    # Check client is active
    clientIsActive=$(cat client.json | jq -r '."'"${client}"'".active')
    echo "${client} active state: ${clientIsActive}"
    if [[ ${clientIsActive} == false ]]; then
        echo "${client} active state is ${clientIsActive} - not scheduling tests."
        return
    fi

    schedule=$(echo ${testSchedule} | jq .schedule)
    echo ${schedule}

    numTests=$(echo ${testSchedule} | jq '.schedule | length')
    echo "${numTests} test(s) for ${client}:"
    echo ${schedule}
    declare -i maxTestSeq=${numTests}-1
    for i in $(seq 0 ${maxTestSeq}); do
        testHour=$(echo ${schedule} | jq .[$i].test_hour)
        thisTest=$(echo ${schedule} | jq .[$i].tests[0])
        # For now only one test in tests - TODO parallel tests
        if [[ ${testHour} == ${scheduleHour} ]]; then
            TEST=$(echo ${thisTest} | jq .test | sed "s/\"//g")
            echo ${TEST}
            TEST_TEMPLATE=$(echo ${thisTest} | jq .test_template | sed "s/\"//g")
            echo TEST_TEMPLATE=${TEST_TEMPLATE}
            enable=$(echo ${thisTest} | jq .enable)
            echo enable=${enable}
            if [[ ${enable} == true ]]; then
                if [[ ${TEST_TEMPLATE} == "" || ${TEST} == "" ]]; then
                    echo "Missing test and/or test template.  Unable to schedule test."
                else
                    echo "******** Triggering test ${TEST} - ${TEST_TEMPLATE} ********"
                    runTestSuite
                    echo
                fi
            else
                echo "Test ${TEST} is disabled."
                echo
            fi
        else
            echo "Not scheduling job.  To be schecduled at ${testHour} hour"
            echo
        fi
    done
}

# Set WORKSPACE if not running in Jenkins
#WORKSPACE= < workspace >
cd ${WORKSPACE}/armada-performance-data/automation

testSchedules=$(cat schedule.json | jq '.'${dayOfWeek}'')
numTestSchedules=$(cat schedule.json | jq '.'${dayOfWeek}' | length')
declare -i maxScheduleSeq=${numTestSchedules}-1

for i in $(seq 0 ${maxScheduleSeq}); do
    scheduleTests $i
done

cd -
