#!/bin/bash -e

# Parse jenkins automation jobs and generate easy to navigate display

# TODO: It would be helpful to pull a list of test names from ./Run-Performance-Tests/config.xml/config.xml and include in output so tests could be added dynamically.

builds=0

createResultsHeader() {
    nowEpocSeconds=$(date +%s)
    lastWeekEpocSeconds=$((nowEpocSeconds-604800))
    lastWeekEpoc=$((lastWeekEpocSeconds*1000))
    echo ${lastWeekEpoc}
    echo -n "{\"timestamp\": $((nowEpocSeconds*1000)), \"builds\": [" > /tmp/parseJenkinsResults.builds.json
}

createResultsTail() {
    echo "]}" >> /tmp/parseJenkinsResults.builds.json
}

getJobsFromJenkins() {
    baseUrl=$1
    jenkinsToken=$2
    jobType=$3
    folders=( $baseUrl )

    set +e

    buildUrls=( $(curl -s --user armada.performance@uk.ibm.com:${jenkinsToken} --data-urlencode "tree=allBuilds[number,url]{0,200}" --data-urlencode "pretty=true" ${baseUrl}/api/json | jq '.allBuilds[] | .url' | sed -e   "s/\"//g") )
    for buildUrl in "${buildUrls[@]}"; do
        if [[ ${jobType} == "default" ]]; then
            # If a new jenkins job parameter is added to this list, 
            #   and not all jobs pulled by running this script have that new parameter, 
            #   then the result will be that no builds are output.
            # TODO It sould be nice to merge the check for AUTOMATED_RUN and grabbing the data for non-manual runs. Probably can now that all builds have AUTOMATED_RUN.
            automation_run=$(curl -s --user armada.performance@uk.ibm.com:${jenkinsToken} --data-urlencode "tree=actions[parameters[name,value]],id,result,description,number,url,timestamp" --data-urlencode "pretty=true" ${buildUrl}/api/json | jq '.actions[] | select(._class == "hudson.model.ParametersAction") | .parameters[] | select(.name == "AUTOMATED_RUN") | .value' | sed -e 's/\"//g')
            if [[ ${automation_run} == "true" ]]; then 
                # Each value from paramaters is pulled with a statement like this:
                #    cluster_prefix: .actions[] | select(._class == "hudson.model.ParametersAction") | .parameters[] | select(.name == "CLUSTER_PREFIX") | .value,
                buildData=$(curl -s --user armada.performance@uk.ibm.com:${jenkinsToken} --data-urlencode "tree=actions[parameters[name,value]],id,result,description,number,url,timestamp" --data-urlencode "pretty=true" ${buildUrl}/api/json | jq '{type: "default", result: .result, description: .description, url: .url, number: .number, timestamp: .timestamp, cluster_prefix: .actions[] | select(._class == "hudson.model.ParametersAction") | .parameters[] | select(.name == "CLUSTER_PREFIX") | .value, cluster_type: .actions[] | select(._class == "hudson.model.ParametersAction") | .parameters[] | select(.name == "CLUSTER_TYPE") | .value, perf_clients: .actions[] | select(._class == "hudson.model.ParametersAction") | .parameters[] | select(.name == "PERF_CLIENTS") | .value, k8s_version: .actions[] | select(._class == "hudson.model.ParametersAction") | .parameters[] | select(.name == "K8S_VERSION") | .value, worker_type: .actions[] | select(._class == "hudson.model.ParametersAction") | .parameters[] | select(.name == "WORKER_TYPE") | .value, cloud_environment: .actions[] | select(._class == "hudson.model.ParametersAction") | .parameters[] | select(.name == "CLOUD_ENVIRONMENT") | .value, zones: .actions[] | select(._class == "hudson.model.ParametersAction") | .parameters[] | select(.name == "ZONES") | .value, perf_test: .actions[] | select(._class == "hudson.model.ParametersAction") | .parameters[] | select(.name == "PERF_TESTS") | .value, delete_cluster: .actions[] | select(._class == "hudson.model.ParametersAction") | .parameters[] | select(.name == "DELETE_CLUSTER") | .value, operating_system: .actions[] | select(._class == "hudson.model.ParametersAction") | .parameters[] | select(.name == "OPERATING_SYSTEM") | .value,}' | sed -e 's/\\\"//g')
                #automated_run: .actions[] | select(._class == "hudson.model.ParametersAction") | .parameters[] | select(.name == "AUTOMATED_RUN") | .value, 

                result=$(echo ${buildData} | jq '.result' | sed -e 's/^"//' -e 's/"$//')
                timestamp=$(echo ${buildData} | jq '.timestamp' | sed -e 's/^"//' -e 's/"$//')
                perf_test=$(echo ${buildData} | jq '.perf_test' | sed -e 's/^"//' -e 's/"$//')

                #echo "|${result}|${timestamp}|${perf_test}|"
                # filter out ones still running, and that are no more than a week old
                if [[ ${result} != "null" && ${timestamp} -ge ${lastWeekEpoc} ]]; then
                    #echo "BUILD: $buildUrl
                    #echo $buildData

                    clusterPrefix=$(echo ${buildData} | jq '.cluster_prefix' | sed -e 's/^"//' -e 's/"$//')
                    if [[ ${clusterPrefix}* == "Perf"* ]]; then
                        # If failure see if it is due to broken pipe
                        if [[ ${result} == "FAILURE" ]]; then
                            broken_pipe=$(curl -s --user armada.performance@uk.ibm.com:${jenkinsToken} --data-urlencode "tree=actions[parameters[name,value]],id,result,description,number,url,timestamp" --data-urlencode              "pretty=true" ${buildUrl}/consoleText | grep "port 22: Broken pipe")
                            echo "Broken pipe: ${broken_pipe}"
                            if [[ ${broken_pipe} == *"Broken pipe"* ]]; then
                                echo "Broken pipe: yes"
                                buildData=$(echo ${buildData} | sed -e "s/FAILURE/FAILURE_BROKEN_PIPE/g")
                            fi
                        fi

                        if [[ ${builds} -ne 0 ]]; then
                            echo -n "," >> /tmp/parseJenkinsResults.builds.json
                        fi
                        echo ${buildData}  >> /tmp/parseJenkinsResults.builds.json
                        builds=$((builds+1))
                    fi
                fi
            fi
        else
            # Assume "registry"
            buildData=$(curl -s --user armada.performance@uk.ibm.com:${jenkinsToken} --data-urlencode "tree=actions[parameters[name,value]],id,result,description,number,url,timestamp" --data-urlencode "pretty=true" ${buildUrl}/api/json | jq '{type: "registry", result: .result, description: .description, url: .url, number: .number, timestamp: .timestamp, 
            perf_clients: .actions[] | select(._class == "hudson.model.ParametersAction") | .parameters[] | select(.name == "PERF_CLIENTS") | .value}' | sed -e 's/\\\"//g')

            result=$(echo ${buildData} | jq '.result' | sed -e 's/^"//' -e 's/"$//')
            timestamp=$(echo ${buildData} | jq '.timestamp' | sed -e 's/^"//' -e 's/"$//')
            perf_test=$(echo ${buildData} | jq '.perf_test' | sed -e 's/^"//' -e 's/"$//')

            #echo "|${result}|${timestamp}|${perf_test}|"
            # filter out ones still running, those that only create the cluster, and that are no more than a week old
            if [[ ${result} != "null" && ${timestamp} -ge ${lastWeekEpoc} && ${perf_test} != "" ]]; then
                #echo "BUILD: $buildUrl
                #echo $buildData

                if [[ ${builds} -ne 0 ]]; then
                    echo -n "," >> /tmp/parseJenkinsResults.builds.json
                fi
                echo ${buildData}  >> /tmp/parseJenkinsResults.builds.json
                builds=$((builds+1))
            fi
        fi
    done
    set -e
}

createResultsHeader
# Don't put '/' on end of url
getJobsFromJenkins "https://alchemy-testing-jenkins.swg-devops.com/job/Armada-performance/job/Automation/job/Run-Performance-Tests" ${STAGE_GLOBAL_ARMPERF_TEST_JENKINS_TOKEN} "default"
getJobsFromJenkins "https://alchemy-testing-jenkins.swg-devops.com/job/Armada-performance/job/Automation/job/Run_registry_tests" ${STAGE_GLOBAL_ARMPERF_TEST_JENKINS_TOKEN} "registry"
createResultsTail

echo "status=${builds} builds were output" > /tmp/buildStatus
exit 0
