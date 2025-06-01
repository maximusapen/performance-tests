#!/bin/bash -e

# Send message to slack channel
# For testing, you can DM yourself by changing channel, e.g. #armada-perf-metrics to @<your slack id>

channel=$1
message="$2"

echo ""
echo "Sending message:"
echo "${message}"
echo "to slack ${channel}"
curl -X POST --data-urlencode "payload={\"channel\": \"${channel}\", \"username\": \"webhookbot\", \"text\": \"${message}\"}" https://hooks.slack.com/services/T4LT36D1N/B01KW68CKPD/${STAGE_GLOBAL_ARMPERF_SLACKTOKEN}
