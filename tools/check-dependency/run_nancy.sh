#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# run Sonatype Nancy tool to identify vulnerabilities in go packages
# (https://github.com/sonatype-nexus-community/nancy)
#
# Script will identify vulnerabilities in go dependencies and post to slack if any are found

# Send message to slack
function sendToSlack() {
    slackText="$1"
    # For testing, you can DM yourself by changing channel from #armada-perf-bots to @<your slack id>
    slackChannel="#armada-perf-bots"
    echo ""
    echo "Sending to slack channel ${slackChannel}"
    echo "${slackText}"
    curl -X POST --data-urlencode "payload={\"channel\": \"${slackChannel}\", \"username\": \"webhookbot\", \"text\": \"${slackText}\", \"icon_emoji\": \":ghost:\"}" https://hooks.slack.com/services/T4LT36D1N/B01KW68CKPD/${STAGE_GLOBAL_ARMPERF_SLACKTOKEN}
}
#Install nancy
# Mac
#curl -L -o ./nancy https://github.com/sonatype-nexus-community/nancy/releases/download/v1.0.29/nancy-v1.0.29-darwin-amd64

# Linux
curl -L -o ./nancy https://github.com/sonatype-nexus-community/nancy/releases/download/v1.0.29/nancy-v1.0.29-linux-amd64
chmod 755 ./nancy

dependency_file="dependency-output.json"
# Run the dependency check
go list -json -m all | ./nancy sleuth -o json-pretty >${dependency_file} 2>&1
cat ${dependency_file}

cat ${dependency_file} | jq empty
if [[ $? -ne 0 ]]; then
    message="ERROR Dependency check did not return valid JSON, check VA_Images job at ${BUILD_URL} for details"
    echo ${message}
    sendToSlack "${message}"
    exit 1
fi
num_vulnerable=$(cat dependency-output.json | jq '.num_vulnerable')
num_audited=$(cat dependency-output.json | jq '.num_audited')
if [ "${num_audited}" -lt "10" ]; then
    message="WARNING Dependency check found less than 10 dependencies to check, there may be a problem, check VA_Images job at ${BUILD_URL} for details"
    echo ${message}
    sendToSlack "${message}"
    exit 1
fi

rm ${dependency_file}
if [ "${num_vulnerable}" -gt "0" ]; then
    slackFile=slack.txt
    echo "\`\`\`" >${slackFile}
    echo "Found ${num_vulnerable} vulnerabilities in go dependencies - check VA_Images job at ${BUILD_URL} for details" >>${slackFile}
    echo "\`\`\`" >>${slackFile}

    # Send to slack channel
    slackText=$(cat ${slackFile})
    echo ${slackText}
    sendToSlack "${slackText}"
    rm ${slackFile}
else
    message="No Vulnerabilities found in go dependencies"
    echo ${message}
    sendToSlack "${message}"
fi
rm ./nancy
