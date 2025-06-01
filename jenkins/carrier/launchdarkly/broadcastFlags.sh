#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018, 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Do not set -x as we do not want to expose STAGE_GLOBAL_ARMPERF_LD_APIKEY
# Console output is also easier to read without -x when trying to find which
# micro-services is updated

upgCarrier=$1
updatedKeys=$2
failUpdatedKeys=$3
broadcastType=$4

echo "upgCarrier: ${upgCarrier}"
echo "updatedKeys: ${updatedKeys}"
echo "failUpdatedKeys: ${failUpdatedKeys}"

# Option --trim to only report updated micro-services, not all
if [[ ${broadcastType} == "--trim" ]]; then
    trim=true
else
    trim=false
fi

echo "Report updated carrier config for $upgCarrier: $updatedKeys"
# getCarrierLDFlags.sh will create a /tmp/${upgCarrier}.txt with current micro-services versions from LD
./getCarrierLDFlags.sh ${upgCarrier} no-reference
echo "updatedKeys: ${updatedKeys}"
cp /tmp/${upgCarrier}.txt /tmp/${upgCarrier}-upgrade.txt

for key in ${updatedKeys}; do
    echo "processing updated ${key}"
    # sed to apply to the first word for the line with * to mark upgrade
    cat /tmp/${upgCarrier}-upgrade.txt | sed "s/^${key}:/ * ${key}:/" >/tmp/${upgCarrier}-tmp.txt
    cp /tmp/${upgCarrier}-tmp.txt /tmp/${upgCarrier}-upgrade.txt
done

for key in ${failUpdatedKeys}; do
    echo "processing fail update ${key}"
    # sed to apply to the first word for the line with - to mark fail upgrade
    cat /tmp/${upgCarrier}-upgrade.txt | sed "s/^${key}:/ - ${key}:/" >/tmp/${upgCarrier}-tmp.txt
    cp /tmp/${upgCarrier}-tmp.txt /tmp/${upgCarrier}-upgrade.txt
done

# Group all updated micro-services in a text box
if [[ ${updatedKeys} != "" ]]; then
    echo "\`\`\`" >/tmp/${upgCarrier}-shuffled.txt
    grep "*" /tmp/${upgCarrier}-upgrade.txt >>/tmp/${upgCarrier}-shuffled.txt
    echo "\`\`\`" >>/tmp/${upgCarrier}-shuffled.txt
fi

if [[ ${failUpdatedKeys} != "" ]]; then
    echo "Processing failUpdatedKeys: ${failUpdatedKeys}"

    # Send failed update list to armada-perf-bots channel
    echo "Failed to upgrade these micro-services for ${upgCarrier}.  Please rebuild job <${BUILD_URL}|here>." >/tmp/${upgCarrier}-failNotify.txt
    echo "\`\`\`" >>/tmp/${upgCarrier}-failNotify.txt
    grep " - " /tmp/${upgCarrier}-upgrade.txt >>/tmp/${upgCarrier}-failNotify.txt
    echo "\`\`\`" >>/tmp/${upgCarrier}-failNotify.txt

    # Also include in notification for armada-perf-metrics
    cat /tmp/${upgCarrier}-failNotify.txt >>/tmp/${upgCarrier}-shuffled.txt

    echo ""
    echo "Sending failed updated keys to slack #armada-perf-bots"
    failNotify=$(cat /tmp/${upgCarrier}-failNotify.txt)
    echo
    curl -X POST --data-urlencode "payload={\"channel\": \"#armada-perf-bots\", \"username\": \"webhookbot\", \"text\": \"${failNotify}\", \"icon_emoji\": \":ghost:\"}" https://hooks.slack.com/services/T4LT36D1N/B01KW68CKPD/${STAGE_GLOBAL_ARMPERF_SLACKTOKEN}

fi

if [[ ${trim} == false ]]; then
    bomKeys=$(cat /tmp/bomKeys.txt)
    echo "Checking bom key list:"
    echo ${bomKeys}
    # Add versions for all other non-deployed micro-services if not trim version
    echo "Other micro-services levels:" >>/tmp/${upgCarrier}-shuffled.txt
    for key in ${bomKeys}; do
        echo "Processing bom key: ${key}"

        onUpdateList=false

        for updatedKey in ${updatedKeys}; do
            if [[ ${key} == ${updatedKey} ]]; then
                onUpdateList=true
                echo "  key is updated."
                break
            fi
        done

        for failUpdatedKey in ${failUpdatedKeys}; do
            if [[ ${key} == ${failUpdatedKey} ]]; then
                onUpdateList=true
                echo "  key failed to get updated."
                break
            fi
        done

        if [[ ${onUpdateList} == false ]]; then
            # Key is not updated, add to Other list
            fullKeyValue=$(grep "^${key}:" /tmp/${upgCarrier}.txt)
            if [[ ${fullKeyValue} == "" ]]; then
                # key exist in bom but is null in LaunchDarkly
                fullKeyValue="$key: null"
            fi
            echo "  key is not updated: ${fullKeyValue}"
            echo "\`${fullKeyValue}\`" >>/tmp/${upgCarrier}-shuffled.txt
        fi
    done
fi

# Add date to notification
echo "$(echo ${upgCarrier}) (* recently updated $(date)):" >/tmp/micro-service-levels.txt

echo "" >>/tmp/micro-service-levels.txt
cat /tmp/${upgCarrier}-shuffled.txt >>/tmp/micro-service-levels.txt

microServiceLevelsSlack="$(cat /tmp/micro-service-levels.txt)"

# Send data to slack channel #armada-perf-metrics
# For testing, you can DM yourself by changing channel from #armada-perf-metrics to @<your slack id>
echo ""
echo "Sending micro-service levels to slack #armada-perf-metrics"
cat /tmp/micro-service-levels.txt
echo
curl -X POST --data-urlencode "payload={\"channel\": \"#armada-perf-metrics\", \"username\": \"webhookbot\", \"text\": \"$microServiceLevelsSlack\", \"icon_emoji\": \":ghost:\"}" https://hooks.slack.com/services/T4LT36D1N/B01KW68CKPD/${STAGE_GLOBAL_ARMPERF_SLACKTOKEN}
