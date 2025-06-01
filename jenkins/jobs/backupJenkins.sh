#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018, 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Backup Jenkins jobs with a GIT branch ready to create PR
# This script is handling failure with slack notification so not using set -e

getConfigFromJenkins() {
    baseBackupDir=$1
    baseUrl=$2
    jenkinsToken=$3
    folders=($baseUrl)
    folderIndex=0
    folderCnt=${#folders[@]}

    set +e

    while (($folderIndex < $folderCnt)); do
        folder=${folders[folderIndex]}
        if [[ ${folder} == *"zArchive"* ]]; then
            ((folderIndex++))
            continue
        fi
        echo Processing folders and jobs in ${folder}

        folders=(${folders[@]} $(curl -s --user armada.performance@uk.ibm.com:${jenkinsToken} ${folder}/api/json?pretty=true | jq '.jobs[] | select(._class == "com.cloudbees.hudson.plugins.folder.Folder") | .url' | sed -e "s/\"//g"))
        folderCnt=${#folders[@]}
        jobUrls=($(curl -s --user armada.performance@uk.ibm.com:${jenkinsToken} ${folder}/api/json?pretty=true | jq '.jobs[] | select(._class == "hudson.model.FreeStyleProject" or ._class == "org.jenkinsci.plugins.workflow.job.WorkflowJob") | .url' | sed -e "s/\"//g"))

        for jobUrl in "${jobUrls[@]}"; do
            backup=${baseBackupDir}/$(echo "$jobUrl" | sed -e "s#$baseUrl##g" -e "s#/job/#/#g" -e 's#/$##g' -e 's#^/##g').xml
            if [[ "$backup" == null ]]; then
                continue
            fi
            if [[ ! -f "${backup}" ]]; then
                backupPath=$(dirname "${backup}")
                mkdir -p "${backupPath}"
                echo "${backup}" >>/tmp/newJobs.txt
            fi
            echo "Backing up: $jobUrl"
            echo "        to: $backup"
            curl -s --user armada.performance@uk.ibm.com:${jenkinsToken} ${jobUrl}/config.xml >${backup}
        done
        ((folderIndex++))
    done
    set -e
}

# Jenkins create a headless branch if cloning from Jenkins.
# Need to do git clone of armada-performance-data here so we can do "git push origin master"
cd ${WORKSPACE}
git clone git@github.ibm.com:alchemy-containers/armada-performance-data.git

cd ${WORKSPACE}/armada-performance-data

# Install pre-commit hook
pip install pre-commit
pre-commit install

# Prepare to backup jenkins jobs
mkdir -p jenkins/jobs
cd jenkins/jobs

# Don't put '/' on end of url
getConfigFromJenkins alchemy-containers-jenkins "https://alchemy-containers-jenkins.swg-devops.com/job/Armada-Performance" ${STAGE_GLOBAL_ARMPERF_CONT_JENKINS_TOKEN}
getConfigFromJenkins alchemy-testing-jenkins "https://alchemy-testing-jenkins.swg-devops.com/job/Armada-performance" ${STAGE_GLOBAL_ARMPERF_TEST_JENKINS_TOKEN}

cd ${WORKSPACE}/armada-performance-data
git add .

changes=$(git diff HEAD)
echo "changes:"
echo ${changes}

if [[ -z ${changes} ]]; then
    echo No changes. Exiting now.
    echo "status=No_change" >/tmp/buildStatus
    exit 0
fi

# Init timestamp
timestamp=$(date +"%Y%m%d_%H%M%S")

# Set +e here so we handle GIT failure with slack message rather than failing the build right after the GIT command
set +e

# Commit all changes from backupJenkins.sh
echo Commit with message "Backup Jenkins jobs ${timestamp}"
git commit -m "Backup Jenkins jobs ${timestamp}"
GIT_COMMIT_RESULT=$?
echo "GIT_COMMIT_RESULT: ${GIT_COMMIT_RESULT}"

if [[ ${GIT_COMMIT_RESULT} == 1 ]]; then
    # Check for pre-commit failure.  Check to see whether it is only baseline version changes.  Commit baseline if no job involved.
    if [[ $(grep "jenkins/jobs" .secrets.baseline) == "" ]]; then
        echo "No Jenkins backup file reported.  Pre-commit changes."
        echo "Add .secrets.baseline and .pre-commit-config.yaml to commit after version update"
        # Update .secrets.baseline with any version changes.  Installing pre-commit will install detect-secrets.
        detect-secrets scan --all-files --update .secrets.baseline

        # Update .pre-commit-config.yaml
        pre-commit clean
        pre-commit gc
        pre-commit autoupdate

        git add .secrets.baseline
        git add .pre-commit-config.yaml

        echo Commit again with message "Backup Jenkins jobs ${timestamp}"
        git commit -m "Backup Jenkins jobs ${timestamp}"
        GIT_COMMIT_RESULT=$?
        echo "GIT_COMMIT_RESULT: ${GIT_COMMIT_RESULT}"
    fi
fi

if [[ ${GIT_COMMIT_RESULT} == 0 ]]; then
    # GIT_COMMIT_RESULT=0 and Commit has changes.  Push the master branch.
    echo Push commit changes to armada-performance-data master
    git push origin master
    GIT_RESULT=$?
    if [[ ${GIT_RESULT} == 0 ]]; then
        GIT_REV=$(git rev-parse HEAD)
        echo "GIT_REV: ${GIT_REV}"
    else
        TEXT="Failed to push backups of Jenkins jobs with error: ${GIT_RESULT}"
        echo ${TEXT}
    fi
else
    TEXT="Failed to commit backups of Jenkins jobs with error: ${GIT_COMMIT_RESULT}"
    echo ${TEXT}
    GIT_RESULT=${GIT_COMMIT_RESULT}
fi

# Prepare slack message
if [[ $GIT_RESULT == "0" ]]; then
    TEXT="Jenkins jobs backed up to armada-performance-data <https://github.ibm.com/alchemy-containers/armada-performance-data/commit/${GIT_REV}|here>!   Handled by <${BUILD_URL}|backup job>."
    echo "status=${GIT_REV}" >/tmp/buildStatus
else
    TEXT="${TEXT}: Check <${BUILD_URL}|backup job>."
    echo "status=error_${GIT_RESULT}" >/tmp/buildStatus
fi

# Send data to slack channel #armada-perf-bots
# For testing, you can DM yourself by changing channel from #armada-perf-bots to @<your slack id>
channel="#armada-perf-bots"
# Sending result to slack
../../automation/bin/sendSlackMessage.sh ${channel} "${TEXT}"

exit $GIT_RESULT
