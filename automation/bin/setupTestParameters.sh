#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2020, 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#

# Set up test parameters as environment variables using template and test files
# from https://github.ibm.com/alchemy-containers/armada-performance-data/tree/master/automation
# Test parameters are also written to temp files in /tmp:
#   <test-template-file>-<test-file>-for-groovy with env values wrapped with quotes for Jenkins pipeline
#   <test-template-file>-<test-file> with no quotes around env values for Jenkins freestyle project

testClient=$1
testTemplate=$2
testFile=$3

if [ $# -lt 3 ]; then
    echo "Usage:"
    echo "    ./setupTestParameters.sh <perf-client> <test-template-file> <test-file>"
    echo "Example:"
    echo "    ./setupTestParameters.sh stage-dal09-perf4-client-01 testsuite-template kube-1.22"
    testTemplate=${TEST_TEMPLATE}
    testFile=${TEST}
    exit 1
fi

# Uncomment and set your WORKSPACE here to test on your mac and debug the script with command like:
#     ./setupTestParameters.sh testsuite-template openshift-default-vpc-gen2
#WORKSPACE=< Your WORKSPACE with armada-performance and armada-performance-data GIT >

echo "WORKSPACE is set up to ${WORKSPACE}"

testFileDir=${WORKSPACE}/armada-performance-data/automation

if [[ ! -f "${testFileDir}/${testTemplate}" ]]; then
    echo
    echo "Error: ${testTemplate} not found in ${testFileDir}"
    echo
    exit 1
fi

if [[ ! -f "${testFileDir}/${testFile}" ]]; then
    echo
    echo "Error: ${testFile} not found in ${testFileDir}"
    echo
    exit 1
fi

echo "Compiling test parameter with ${testTemplate} and ${testFile}"

declare -a allParm=()

# Process test-template file and add parameters
while read LINE; do
    if [[ ${LINE} == "#"* ]]; then
        # Comment line
        continue
    fi
    parm=$(echo $LINE | sed "s/=/ /" | awk '{print $1}')
    echo "Add ${parm}"
    allParm+=("${parm}")
    export "$LINE"
done <${testFileDir}/${testTemplate}

# Process test file which may add or override parameters from template file
while read LINE; do
    if [[ ${LINE} == "#"* ]]; then
        # Comment line
        continue
    fi
    parm=$(echo $LINE | sed "s/=/ /" | awk '{print $1}')
    if [[ "${allParm[@]}" =~ "${parm}" ]]; then
        echo "Override ${parm}"
    else
        echo "Add ${parm}"
        allParm+=("${parm}")
    fi
    export "$LINE"

done <${testFileDir}/${testFile}

# Start with PERF_TESTS
testsToRun=${PERF_TESTS}
echo "testsToRun: ${testsToRun}"

# Exclude tests in EXCLUDE_TESTS

echo "Excluding tests in EXCLUDE_TESTS: ${EXCLUDE_TESTS}"
excludeTestList=$(echo ${EXCLUDE_TESTS} | sed "s/,/ /g" | sed "s/\"//g")
echo "excludeTestList: ${excludeTestList}"
for test in ${excludeTestList}; do
    echo "Excluding ${test}"
    modifiedTests=$(echo ${testsToRun} | sed "s/${test}//" | sed "s/,,/,/g" | sed "s/,\"/\"/")
    testsToRun=${modifiedTests}
    echo "testsToRun: ${testsToRun}"
done

echo
echo "testsToRun: ${testsToRun}"
echo

# Exclude tests in PERF_TESTS_CLIENT_01 if client is not client-01
if [[ ${testClient} != *"client-01"* ]]; then
    echo
    echo "Excluding tests in PERF_TESTS_CLIENT_01 for ${testClient}: ${PERF_TESTS_CLIENT_01}"
    excludeTestList=$(echo ${PERF_TESTS_CLIENT_01} | sed "s/,/ /g" | sed "s/\"//g")
    echo "excludeTestList: ${excludeTestList}"
    for test in ${excludeTestList}; do
        modifiedTests=$(echo ${testsToRun} | sed "s/${test}//" | sed "s/,,/,/g" | sed "s/,\"/\"/")
        testsToRun=${modifiedTests}
        echo "testsToRun: ${testsToRun}"
    done
fi

echo "testsToRun: ${testsToRun}"
export TESTS_TO_RUN=${testsToRun}
allParm+=("TESTS_TO_RUN")

# Check paremeters set in env and write out files for Jenkins jobs
# envFile for Run-Performance-TestSuite - a pipeline project
# and requires file in groovy format with values in quotes
envFile="/tmp/${testTemplate}-${testFile}-for-groovy"
# envFile2 for Schedule-Performance-TestSuite* jobs - shell scripts
envFile2="/tmp/${testTemplate}-${testFile}"
rm -f ${envFile}
rm -f ${envFile2}
echo
echo "Writing parameters in ${envFile} and ${envFile2}"
echo
echo "allParm: ${allParm[@]}"
echo
for parm in "${allParm[@]}"; do
    parmValue=$(env | grep ${parm}= | sed "s/${parm}=//")
    echo "${parm}=${parmValue}" >>${envFile}
    parmValue2=$(echo ${parmValue} | sed "s/\"//g")
    echo "${parm}=${parmValue2}" >>${envFile2}
done

echo "----------------- ${envFile} contents: -----------------"
cat ${envFile}
echo "---------------------------------------------------------"
echo
echo "----------------- ${envFile2} contents: -----------------"
cat ${envFile2}
echo "--------------------------------------------------------"
