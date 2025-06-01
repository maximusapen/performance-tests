#!/bin/bash -e

# Do not set bash -e so we handle GIT failure with slack message rather than failing the script right after the GIT command

# Merge changes to armada-performance-data in
# https://github.ibm.com/alchemy-containers/armada-performance-data

# The repository needs to be cloned in script as

# (1) Clone armada-performance-data in script and not in Jenkins as
#     Jenkins create a headless branch:
#         git clone git@github.ibm.com:alchemy-containers/armada-performance-data.git
# (2) Make the change to be merged
# (3) cd to the directory where changes in directory and sub-directories are to be merged
# (4) Call this script:
#         ./mergeArmadaPerformanceData.sh < commit message >

commitMessage=$1

gitMergeStatusFile=${WORKSPACE}/mergeStatus.log

git add .

changes=$(git diff HEAD)
echo "changes:"
echo ${changes}

if [[ -z ${changes} ]]; then
    echo "No changes. Exiting now."
    echo "status=No_change" >${gitMergeStatusFile}
    exit 0
fi

# Init timestamp
timestamp=$(date +"%Y%m%d_%H%M%S")

# Commit all changes
echo Commit with message "${commitMessage} ${timestamp}"
git commit -m "${commitMessage} ${timestamp}"
declare -i GIT_COMMIT_RESULT=$?
echo "GIT_COMMIT_RESULT: ${GIT_COMMIT_RESULT}"
declare -i GIT_RESULT=${GIT_COMMIT_RESULT}

TEXT=${commitMessage}

if [[ ${GIT_COMMIT_RESULT} == 0 ]]; then
    # GIT_COMMIT_RESULT=0 and Commit has changes.  Push the master branch.
    echo Push commit changes to armada-performance-data master
    git push origin master
    declare -i GIT_RESULT=$?
    if [[ ${GIT_RESULT} == 0 ]]; then
        GIT_REV=$(git rev-parse HEAD)
        echo "GIT_REV: ${GIT_REV}"
    else
        TEXT="${TEXT}.  Failed to push changes with error: ${GIT_RESULT}"
        echo ${TEXT}
    fi
else
    TEXT="${TEXT}.  Failed to commit changes with error: ${GIT_COMMIT_RESULT}"
    echo ${TEXT}
fi

if [[ $GIT_RESULT == 0 ]]; then
    slackText="${TEXT}: Committed armada-performance-data <https://github.ibm.com/alchemy-containers/armada-performance-data/commit/${GIT_REV} | here>!   Handled by <${BUILD_URL} | Jenkins job>."
    echo "Message=\"${TEXT}\"" >${gitMergeStatusFile}
    echo "status=${GIT_REV}" >>${gitMergeStatusFile}
else
    slackText="${TEXT}: Check <${BUILD_URL}|Jenkins job>."
    echo "Message=\"${TEXT}\"" >${gitMergeStatusFile}
    echo "status=error_${GIT_RESULT}" >>${gitMergeStatusFile}
fi

# Send GIT merge result to slack channel #armada-perf-private
# For testing, you can DM yourself by changing channel from #armada-perf-private to @<your slack id>
slackChannel="#armada-perf-private"
echo "Sending GIT merge result to slack channel ${slackChannel}"
curl -X POST --data-urlencode "payload={\"channel\": \"${slackChannel}\", \"username\": \"webhookbot\", \"text\": \"${slackText}\", \"icon_emoji\": \":ghost:\"}" https://hooks.slack.com/services/T4LT36D1N/B01KW68CKPD/${STAGE_GLOBAL_ARMPERF_SLACKTOKEN}

exit ${GIT_RESULT}
