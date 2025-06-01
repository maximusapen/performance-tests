#!/bin/bash
# Script to run "kubectl rollout restart" of resources in armada namespace.
# Then watch the rollout in watch*.log files.
# Resource can be deployments, daemonsets or statefulsets.
# The rollout should not cause any Jmeter errors.
# Note: Currently there are no statefulsets restart in armada namespace.

if [[ $# -ne 1 ]]; then
    echo "Usage :"
    echo "   rollout.sh  [ deployments | daemonsets ] "
    exit 1
fi

resource=$1

logFile=rollout_${resource}.log
date >${logFile}
kubectl rollout restart ${resource} -n armada >>${logFile}
cat ${logFile}

restartResources=$(cat ${logFile} | awk '{print $1}')
echo "Watching restart for:"
echo ${restartResources}

for watch in ${restartResources}; do
    echo "Watching ${watch}"
    watchFile=watch_$(echo ${watch} | sed "s/\//_/").log
    date >${watchFile}
    kubectl rollout status ${watch} -n armada >>${watchFile} &
done

# You can watch all the rollout with:
#tail -f watch*.log
