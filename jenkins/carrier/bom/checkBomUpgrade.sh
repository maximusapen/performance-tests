#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020, 2022 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

CARRIER_HOST=$1
PerfClient=$2

echo Using Perf Client ${PerfClient} for ${CARRIER_HOST}

# ssh to perf clients frequently timeout.  Retry before failing job.
maxRetry=5
sleepTime=300
set +e
for i in $(seq 1 ${maxRetry}); do
    ssh jenkins@${PerfClient} -o StrictHostKeyChecking=no -o PasswordAuthentication=no "export GOPATH=/performance; export ARMADA_PERFORMANCE_API_KEY=${STAGE_GLOBAL_ARMPERF_IBMCLOUD_APIKEY}; /performance/bin/armada-perf-client2 versions --json" >kubeVersions.json
    if [[ $? == 0 ]]; then
        # ssh was successful
        break
    fi
    # ssh failed.  Retry after sleep.
    sleep ${sleepTime}
done
set -e

if [[ ! -f kubeVersions.json ]]; then
    # Failed to get kubeVersons.json.  Send slack notificaton and exit with error
    channel="#armada-perf-private"
    ../../../automation/bin/sendSlackMessage.sh ${channel} "Failed mark-cruiser-bom-release job for ${CARRIER_HOST} during ssh to performance client. Rebuild job <${BUILD_URL}|here>"
    exit 1
fi

cat kubeVersions.json

jq --version

allDefaultKubeVersions=""
allKubeVersions=""
allTestVersions=""

echo "Checking Kubernetes kube versions"

# Jenkins currently on jq 1.5 which has this parse error for kubeVersions.json from perf client:
#     parse error: Invalid numeric literal at line 56, column 4
# Despite error, the data returned is correct.  jq 1.6 has no error.
# Adding `set +e` and `set -x` for script to continue processiong.
# Do not remove until Jenkins is upgraded to jq 1.6.
set +e

declare i=0
allKubernetesKubeVersions=$(cat kubeVersions.json | jq -j '.kubernetes[] | .major, ".", .minor, " "')
# Example output: allKubernetesKubeVersions: 1.15 1.16 1.17 1.18 1.19
echo "allKubernetesKubeVersions: ${allKubernetesKubeVersions}"
for kubernetesKubeVersion in ${allKubernetesKubeVersions}; do
    # Check for default and deprecated versions
    echo "Checking ${kubernetesKubeVersion}"
    defaultStatus=$(cat kubeVersions.json | jq -j '.kubernetes['$i'] | .default')
    echo "defaultStatus: ${defaultStatus}"
    eosStatus=$(cat kubeVersions.json | jq -j '.kubernetes['$i'] | .end_of_service')
    echo "eosStatus: ${eosStatus}"
    echo "${kubernetesKubeVersion}: default=${defaultStatus}, end_of_service=${eosStatus}"
    if [[ ${defaultStatus} == true ]]; then
        echo
        echo "*** ${kubernetesKubeVersion} is default Kube version for Kubernetes ***"
        echo
        allDefaultKubeVersions="${allDefaultKubeVersions} ${kubernetesKubeVersion}"
    fi
    if [[ ${eosStatus} != "" ]]; then
        echo
        echo "*** ${kubernetesKubeVersion} end of service: ${eosStatus} - Remove from list ***"
        echo
        allKubernetesKubeVersions=$(echo ${allKubernetesKubeVersions} | sed "s/${kubernetesKubeVersion}//g")
    fi
    if [[ ${defaultStatus} == false && ${eosStatus} == "" ]]; then
        # Needs to enable Jenkins testing if updated and not default version
        allTestVersions="${allTestVersions} ${kubernetesKubeVersion}"
    fi
    ((i++))
done

echo
echo "Checking Openshift kube versions"
i=0
allOpenshiftKubeVersions=$(cat kubeVersions.json | jq -j '.openshift[] | .major, ".", .minor, " "')
# Example output: allOpenshiftKubeVersions: 3.11 4.3 4.4
echo "allOpenshiftKubeVersions: ${allOpenshiftKubeVersions}"
for openshiftKubeVersion in ${allOpenshiftKubeVersions}; do
    # Check for default and deprecated versions
    echo "openshiftKubeVersion: ${openshiftKubeVersion}"
    defaultStatus=$(cat kubeVersions.json | jq -j '.openshift['$i'] | .default')
    echo "defaultStatus: ${defaultStatus}"
    eosStatus=$(cat kubeVersions.json | jq -j '.openshift['$i'] | .end_of_service')
    echo "eosStatus: ${eosStatus}"
    echo "${openshiftKubeVersion}: default=${defaultStatus}, end_of_service=${eosStatus}"
    if [[ ${defaultStatus} == true ]]; then
        echo
        echo "*** ${openshiftKubeVersion} is default Kube version for Openshift ***"
        echo
        allDefaultKubeVersions="${allDefaultKubeVersions} ${openshiftKubeVersion}"
    fi
    if [[ ${eosStatus} != "" ]]; then
        echo
        echo "*** ${openshiftKubeVersion} end of service: ${eosStatus} - Remove from list ***"
        echo
        allOpenshiftKubeVersions=$(echo ${allOpenshiftKubeVersions} | sed "s/${openshiftKubeVersion}//g")
    fi
    if [[ ${defaultStatus} == false && ${eosStatus} == "" ]]; then
        # Needs to enable Jenkins testing if updated and not default version
        allTestVersions="${allTestVersions} ${openshiftKubeVersion}"
    fi
    ((i++))
done
set -e

echo "Kubernetes versions to process: ${allKubernetesKubeVersions} ${allOpenshiftKubeVersions}"
echo "Default Kubernetes versions: ${allDefaultKubeVersions}"
echo "Enable Jenkins testing for Kubernetes versions if updated: ${allTestVersions}"

allFullVersions=""
bomFilePrefix="${WORKSPACE}/armada-ansible/common/bom/armada-ansible-bom"
for kubeVersion in ${allKubernetesKubeVersions}; do
    fullKubeVersion=$(grep published_ansible_bom_version ${bomFilePrefix}-${kubeVersion}.yml | awk '{print $2}')
    echo "Full version for ${kubeVersion}:   ${fullKubeVersion}"
    allFullVersions="${allFullVersions} ${fullKubeVersion}"
done

# OpenShift has different bom filenames in armada-ansible
bomFilePrefix="${WORKSPACE}/armada-ansible/common/bom/openshift-target-bom"
for kubeVersion in ${allOpenshiftKubeVersions}; do
    fullKubeVersion=$(grep published_ansible_bom_version ${bomFilePrefix}-${kubeVersion}.yml | awk '{print $2}')
    echo "Full version for ${kubeVersion}:   ${fullKubeVersion}"
    allFullVersions="${allFullVersions} ${fullKubeVersion}"
done

echo "allFullVersions: ${allFullVersions}"

# Check for BOM updates for allFullVersions with previous build description.
curl -s --user armada.performance@uk.ibm.com:${STAGE_GLOBAL_ARMPERF_CONT_JENKINS_TOKEN} --output "mark-cruiser-bom-release.json" https://alchemy-containers-jenkins.swg-devops.com/job/Armada-Performance/job/carrier-pipeline/job/mark-cruiser-bom-release/api/json?depth=2

if [[ ! -f mark-cruiser-bom-release.json ]]; then
    # Failed to get kubeVersons.json.  Send slack notificaton and exit with error
    channel="#armada-perf-private"
    ../../../automation/bin/sendSlackMessage.sh ${channel} "Failed mark-cruiser-bom-release job for ${CARRIER_HOST} when getting previous bom upgrade history. Rebuild job <${BUILD_URL}|here>"
    exit 1
fi

for i in $(seq 0 20); do
    displayName=$(cat mark-cruiser-bom-release.json | jq '.builds['$i'].displayName')
    description=$(cat mark-cruiser-bom-release.json | jq '.builds['$i'].description')
    echo "Build ${displayName}: ${description}"
    if [[ ${description} == *"${CARRIER_HOST}"*"-"* ]]; then
        # Build description with "${CARRIER_HOST} -" (with hyphen) is a good build with all full kube versions listed
        break
    fi
done

echo
allBuildVersion=$(echo ${description} | sed "s/${CARRIER_HOST} - //" | sed 's/"//g')
echo "Found build ${displayName} for ${CARRIER_HOST} with Kube versions:"
for buildVersion in ${allBuildVersion}; do
    echo ${buildVersion}
done
echo

# Compare current full version in allFullVersions with last build for the carrier in allBuildVersion
echo "Checking previous build kube versions with current kube versions on ${CARRIER_HOST}:"
echo ${allFullVersions}
echo
allUpdatedFullVersions=""
for fullVersion in ${allFullVersions}; do
    if [[ ${description} == *"${fullVersion}"* ]]; then
        echo "${fullVersion} has not been changed."
    else
        echo
        echo "*  ${fullVersion} has been updated. *"
        echo
        allUpdatedFullVersions="${allUpdatedFullVersions}   ${fullVersion}"
    fi
done

if [[ ${allUpdatedFullVersions} == '' ]]; then
    echo
    echo "All kube versions are up-to-date with no change on ${CARRIER_HOST}:"
    echo "${description}"
    echo
else
    echo
    echo "Trigger BOM update(s) for: ${allUpdatedFullVersions}"
    for triggerVersion in ${allUpdatedFullVersions}; do
        echo "Triggering performance-armada-bom-preview-state job for $triggerVersion"
        curl -i -s -k -X POST --user armada.performance@uk.ibm.com:${STAGE_GLOBAL_ARMPERF_CONT_JENKINS_TOKEN} "https://alchemy-containers-jenkins.swg-devops.com/job/Containers-Runtime/view/Armada-BOM/job/performance-armada-bom-preview-state/buildWithParameters?BOM_VERSION=${triggerVersion}&PREVIEW_STATE=nil&REGION=stage"
        if [[ $? -ne 0 ]]; then
            # Failed to get kubeVersons.json.  Send slack notificaton and exit with error
            channel="#armada-perf-private"
            ../../../automation/bin/sendSlackMessage.sh ${channel} "Failed mark-cruiser-bom-release job for ${CARRIER_HOST} when triggering performance-armada-bom-preview-state job. Rebuild job <${BUILD_URL}|here>"
            exit 1
        fi
    done
fi

echo "allUpdatedFullVersions: ${allUpdatedFullVersions}"
echo "Default BOMs: ${allDefaultKubeVersions}"
echo "allTestVersions: ${allTestVersions}"

allEnableTestVersions=""

# Clone armada-performance-data before calling enableTest.sh so we can update with "git push origin master"
# Jenkins create a headless branch if cloning from Jenkins.
# Need to do git clone of armada-performance-data here so we can do
git clone git@github.ibm.com:alchemy-containers/armada-performance-data.git

# Get list to enable Jenkins jobs for updated BOMs
slackFile=/tmp/slack.txt
echo "BOM upgrade on \`${CARRIER_HOST}\`:" >${slackFile}
allEnableCommitVersion=""
for updatedFullVersion in ${allUpdatedFullVersions}; do
    echo "\`${updatedFullVersion}\`:" >>${slackFile}
    for defaultKubeVersion in ${allDefaultKubeVersions}; do
        if [[ ${updatedFullVersion} == "${defaultKubeVersion}"* ]]; then
            echo "${updatedFullVersion} is default version"
            echo "  - default" >>${slackFile}
            break
        fi
    done
    for testKubeVersion in ${allTestVersions}; do
        echo "Checking ${testKubeVersion} against ${updatedFullVersion}"
        if [[ ${updatedFullVersion} == "${testKubeVersion}"* ]]; then
            allEnableTestVersions="${allEnableTestVersions} ${testKubeVersion}"

            echo "CARRIER_HOST: ${CARRIER_HOST}"

            if [[ ${allKubernetesKubeVersions} == *${testKubeVersion}* ]]; then
                testfile="kube-${testKubeVersion}"
            elif [[ ${allOpenshiftKubeVersions} == *${testKubeVersion}* ]]; then
                testfile="openshift-${testKubeVersion}"
            fi

            ${WORKSPACE}/armada-performance/automation/bin/enableTest.sh ${CARRIER_HOST} ${testfile} enable ${slackFile}
            enableStatus=$(tail -1 ${slackFile})
            if [[ ${enableStatus} == *"- To enable test"* ]]; then
                enableCommitVersion=$(echo ${enableStatus} | sed "s/- To enable test //")
                allEnableCommitVersion="${allEnableCommitVersion} ${enableCommitVersion}"
            fi
        fi
    done
done

echo "allEnableCommitVersion: ${allEnableCommitVersion}"

if [[ ${allUpdatedFullVersions} != "" ]]; then

    # Send BOM update to slack channel #armada-perf-private
    # For testing, you can DM yourself by changing channel from #armada-perf-private to @<your slack id>
    slackChannel="#armada-perf-private"
    echo ""
    echo "Sending new BOM versions to slack channel ${slackChannel}"
    slackText=$(cat ${slackFile})
    echo ${slackText}
    curl -X POST --data-urlencode "payload={\"channel\": \"${slackChannel}\", \"username\": \"webhookbot\", \"text\": \"${slackText}\", \"icon_emoji\": \":ghost:\"}" https://hooks.slack.com/services/T4LT36D1N/B01KW68CKPD/${STAGE_GLOBAL_ARMPERF_SLACKTOKEN}
    # Update armada-performance-data
    # mergeArmadaPerformanceData.sh will send slack notification on merge status
    cd ${WORKSPACE}/armada-performance-data/automation
    ${WORKSPACE}/armada-performance/automation/bin/mergeArmadaPerformanceData.sh "*** Enable tests for ${allEnableCommitVersion}"
    cd -
fi

echo
echo "All Updated BOMs: ${allUpdatedFullVersions}"
echo "Default BOMs: ${allDefaultKubeVersions}"
echo "Enabling Jenkins for kube versions (non-default): ${allEnableTestVersions}"
echo "allEnableCommitVersion: ${allEnableCommitVersion}"

echo allFullVersions="${allFullVersions}" >fullKubeVersions.properties

cat fullKubeVersions.properties
