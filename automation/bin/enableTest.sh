#!/bin/bash -e

# Enable or disable non-default kube version testings in
# https://github.ibm.com/alchemy-containers/armada-performance-data/tree/master/automation/schedule.json.

# Calling script or Jenkins job responsible for merge/push to update armada-performance-data
# to allow multiple updates before merge/push.

if [ $# -lt 3 ]; then
    echo "Usage:"
    echo "    ./enableTest.sh < specific carrier | all > < test > < enable | disable > [ < path to message File >]"
    echo "Example:"
    echo "    To enable kube-1.18 tests on carrier4 writing to optional message file:"
    echo "        ./enableTest.sh stage-dal10-carrier4 kube-1.18 enable /tmp/slack.txt"
    echo "    To disable kube-1.18 tests on all carriers:"
    echo "        ./enableTest.sh all kube-1.18 disable"
    exit 1
fi

CARRIER_HOST=$1
test=$2
enableText=$3
slackFile=$4

if [[ ${enableText} == 'enable' ]]; then
    enable=true
elif [[ ${enableText} == 'disable' ]]; then
    enable=false
else
    echo "Please specifiy enable or disable for test"
    echo "Usage:"
    echo "    ./enableTest.sh < specific carrier | all > < test > < enable | disable >"
    echo "Example:"
    echo "    To enable kube-1.18 tests on carrier4:"
    echo "        ./enableTest.sh stage-dal10-carrier4 kube-1.18 enable"
    echo "    To disable kube-1.18 tests on all carriers:"
    echo "        ./enableTest.sh all kube-1.18 disable"
    exit 1
fi

scheduleFile=${WORKSPACE}/armada-performance-data/automation/schedule.json

echo "Changing ${test} - enable=${enable} in ${scheduleFile}"

# New
if [[ ${CARRIER_HOST} == "all" ]]; then
    testClients="*"
else
    testClients="stgiks-dal10-perf$(echo ${CARRIER_HOST} | sed "s/-/ /g" | awk '{print $3}' | sed "s/carrier//")"
fi
echo "Clients for ${CARRIER_HOST}: ${testClients}"

declare -a daysOfWeek=(Monday Tuesday Wednesday Thursday Friday Saturday Sunday)

testUpdated=false
testHasSchedule=false
slackMessage=""
for day in ${daysOfWeek[@]}; do
    echo "Checking ${day}"
    clients=$(jq '.'${day}'[].client' ${scheduleFile} | sed "s/\"//g")
    for client in ${clients}; do
        echo "Checking ${client}"
        testFound=$(jq '.'${day}'[] | select(.client=="'${client}'") .schedule | select(.[].tests[].test=="'${test}'")' ${scheduleFile})
        if [[ ${testFound} != "" ]]; then
            echo
            echo Test scheduled on client: ${client}
            echo "Current Schedule:"
            echo ${testFound}
            if [[ ${testClients} == "*" || ${client} == "${testClients}"* ]]; then
                echo "Setting enable to ${enable} for test ${test} on ${client} for ${CARRIER_HOST}"
                newSchedule=$(jq '(.'${day}'[].schedule[].tests[] | select(.test=="'${test}'") .enable) = '${enable}'' ${scheduleFile})
                # Update schedule.json file with new schedule
                echo ${newSchedule} | jq . >${scheduleFile}
                updatedTest=$(jq '.'${day}'[] | select(.client=="'${client}'") .schedule | select(.[].tests[].test=="'${test}'")' ${scheduleFile})
                echo "Updated Schedule:"
                echo ${updatedTest}
                if [[ "${testFound}" == "${updatedTest}" ]]; then
                    message="- Test ${test} is already ${enableText}d"
                else
                    # Do not change format of message below.  Use by checkBomUpgrade.sh for getting enable status
                    message="- To ${enableText} test ${test}"
                fi
                if [[ ${slackMessage} != *${message}* ]]; then
                    # Add to slack message if not already added by previous update
                    slackMessage="${slackMessage} ${message}"
                fi
            else
                testCarrier="stage carrier$(echo ${client} | sed "s/-/ /g" | awk '{print $3}' | sed "s/perf//")"
                message="- Not enabling test ${test}.  Test is scheduled on ${testCarrier}"
                if [[ ${slackMessage} != *${message}* ]]; then
                    # Add to slack message if not already added by previous update
                    slackMessage="${slackMessage} ${message}"
                fi
            fi
        fi
    done
done

if [[ ${slackMessage} == "" ]]; then
    slackMessage="- ${test} has no test schedule"
fi

echo SlackMessage
echo ${slackMessage}

if [[ ${slackFile} != "" ]]; then
    echo ${slackMessage} >>${slackFile}
fi
