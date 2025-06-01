#!/bin/bash

commsPort=4444
mode=slave

# Setup for working with standalone pods
curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl >kubectl

kubectl_path=$(pwd)
PATH=$PATH:$kubectl_path
echo $PATH
chmod +x ./kubectl

# JMeter requires a comma separated list of ips of the slave nodes
# We know how many we should have, so loop until we've got them all.
while
    slave_ips=$(kubectl get pod -o wide --selector app=jmeter-slave | grep " Running " | awk '{print $6}' | sort -u | awk -v ORS=, '{ print $1 }' | sed 's/,$//')
    IFS=',' read -r -a slave_arr <<<"${slave_ips}"
    ((${#slave_arr[@]} != ${SLAVE_PODS}))
do
    printf "%s - Waiting for all slave pods.\n" "$(date +%T)"
    sleep 5
done

while true; do
    # Wait for a test command to be sent by the user via the master-controller pod's nodeport
    printf "%s - Waiting for test command....\n" "$(date +%T)"

    # We'll get the client's ip for audit purposes
    cmd=$(nc -lnvp ${commsPort} 2>connection)
    clientIP=$(sed -nr "s/.*from.*\[([0-9\.]*)].*/\1/p" connection)

    IFS=' ' read -r -a cmd_data <<<"${cmd}"
    case "${cmd_data[0]}" in
    "START")
        printf "%s - \"%s\" received from %s\n" "$(date +%T)" "${cmd}" "${clientIP}"
        if ((${#cmd_data[@]} > 1)); then
            # Override the number of JMeter threads (if specified)
            param="${cmd_data[1]}"
            JMETER_ARGS=$(echo ${JMETER_ARGS} | sed "s/-GTHREAD=[0-9]* //")
            JMETER_ARGS="${JMETER_ARGS} -GTHREAD=${param} "
        fi
        if ((${#cmd_data[@]} > 2)); then
            # Override the test throughput (if specified)
            param=$(awk "BEGIN {print ${cmd_data[2]} * 60}")
            JMETER_ARGS=$(echo ${JMETER_ARGS} | sed "s/-GTPUTMINS=[0-9]* //")
            JMETER_ARGS="${JMETER_ARGS} -GTPUTMINS=${param} "
        fi
        if ((${#cmd_data[@]} > 3)); then
            # Override the test duration (if specified)
            param="${cmd_data[3]}"
            JMETER_ARGS=$(echo ${JMETER_ARGS} | sed "s/-GRUNLENSEC=[0-9]* //")
            JMETER_ARGS="${JMETER_ARGS} -GRUNLENSEC=${param} "
            #olb-java tests uses different param for run length
            JMETER_ARGS=$(echo ${JMETER_ARGS} | sed "s/-GDURATION=[0-9]* //")
            JMETER_ARGS="${JMETER_ARGS} -GDURATION=${param} "
        fi
        if ((${#cmd_data[@]} > 4)); then
            mode="${cmd_data[4]}"
        fi
        if ((${#cmd_data[@]} > 5)); then
            setNamespace="-n ${cmd_data[5]}"
        fi
        ;;
    "STOP" | "EXIT")
        printf "%s - \"%s\" received from %s\n" "$(date +%T)" "${cmd}" "${clientIP}"
        break 2
        ;;
    "RESULTS")
        # Covers the case where sometimes the client fails to get the results that are sent.
        printf "%s - \"%s\" received from %s\n" "$(date +%T)" "${cmd}" "${clientIP}"
        printf "%s - Waiting to send results...\n" "$(date +%T)"
        nc -lp ${commsPort} <results/results.csv
        cat results/results.csv
        continue
        ;;
    *)
        printf "%s - Unrecognised command \"%s\" received. Ignoring.\n" "$(date +%T)" "${cmd}"
        continue
        ;;
    esac

    printf "%s - Starting test.\n" "$(date +%T)"
    printf "Slave pods: %s\n" "${slave_ips}"
    if [[ -d "results" ]]; then
        rm -r results
    fi

    mkdir -p results
    cd results

    # Start tests
    if [[ $mode == "slave" ]]; then
        set -x
        jmeter -n -t ../test.jmx -Dhttpclient.reset_state_on_thread_group_iteration=false -Dserver.rmi.ssl.disable=true -R "${slave_ips}" -f -l samples.out ${JMETER_ARGS}
        set +x

        mv jmeter.log jmeter-master.log

        if [[ ${JMETER_ARGS} == *"mode=Statistical"* ]]; then
            echo "mode=Statistical"
            #SLAVE_PODS
            THREADS=$(echo ${JMETER_ARGS} | sed "s/^.*-GTHREAD=\([0-9]*\) .*$/\1/")
            PUTSMIN=$(echo ${JMETER_ARGS} | sed "s/^.*-GTPUTMINS=\([0-9]*\) .*$/\1/")
            RUNLENSEC=$(echo ${JMETER_ARGS} | sed "s/^.*-GRUNLENSEC=\([0-9]*\).*$/\1/")
            echo "pods,threads per pod,puts per min,duration,dateTime,# Samples,Duration,Throughput,Errors,Error %" >results.csv
            grep "summary = " jmeter-master.log | tail -1 | sed -e "s/^\(.*\),.*summary = \(.*\) in \([0-9:]*\) = \([0-9.]*\).s Avg.*Err: *\([0-9]*\) .*(\(.*%\).*$/${SLAVE_PODS},${THREADS},${PUTSMIN},${RUNLENSEC},\1,\2,\3,\4,\5,\6/g" >>results.csv
            cat results.csv
        else
            # Get results in csv format (this is the standard format used in all armada performance JMeter based tests)
            JMeterPluginsCMD.sh --tool Reporter --plugin-type AggregateReport --input-jtl samples.out --generate-csv results.csv
        fi
    elif [[ $mode == "standalone" ]]; then
        set -x
        now=$(date "+%Y.%m.%d_%H%M%S")
        for ((i = 0; i < ${#slave_arr[@]}; i++)); do
            echo "Sending request to " nc "${slave_arr[i]}" "${commsPort}"
            echo -en "${cmd}" | nc -q 0 "${slave_arr[i]}" "${commsPort}"
        done
        set +x

        printf "%s - Poll for results.\n" "$(date +%T)"

        for ((i = 0; i < ${#slave_arr[@]}; i++)); do
            counter=1
            # Now wait for up to 30 minutes for the JMeter test to complete. A slave listening on port 4444
            # indicates that it has completed the test.
            until [[ $counter -gt 180 ]]; do
                nc -z "${slave_arr[i]}" "${commsPort}"
                if [[ $? -eq 0 ]]; then
                    printf "%s. ${slave_arr[i]} has completed the test.\n" "$(date +%T)"
                    break
                fi
                printf "%s - %d. Waiting for results from ${slave_arr[i]} to become available.\n" "$(date +%T)" "${counter}"
                sleep 20
                ((counter++))
            done
        done

        for pod in $(kubectl ${setNamespace} get pods | grep jmeter-standalone-slave- | awk '{print $1}'); do
            printf "%s - Pulling results from %s.\n" "$(date +%T)" "${pod}"
            # Support retry of temperamental command
            counter=1
            resultsFound=false
            until [[ $counter -gt 5 ]]; do
                kubectl $setNamespace exec ${pod} -- tar czf - -C /jmeter/results/ samples.out | tar xzO >temp.${pod}.samples.out

                if [[ $? -eq 0 ]]; then
                    # Command was successful
                    resultsFound=true
                    break
                fi

                # If command failed this file probably doesn't exist, but remove to be sure before trying again
                rm -f temp.${pod}.samples.out

                sleep 30
                ((counter++))
            done

            if [[ ${resultsFound} == false ]]; then
                break
            fi
        done

        if [[ ${resultsFound} == true ]]; then
            resultsFile="results.csv"
            echo "timeStamp,elapsed,label,responseCode,responseMessage,threadName,dataType,success,failureMessage,bytes,sentBytes,grpThreads,allThreads,URL,Latency,IdleTime,Connect" >samples.out
            cat temp.*.samples.out | grep -v timeStamp >>samples.out
            rm -rf temp.*.samples.out
            printf "%s - First 20 entries in samples.out: .\n" "$(date +%T)"
            head -n 20 samples.out

            # Get results in csv format (this is the standard format used in all armada performance JMeter based tests)
           JMeterPluginsCMD.sh --tool Reporter --plugin-type AggregateReport --input-jtl samples.out --generate-csv ${resultsFile}

           # Send results file back to the test initiator
           # Might have to wait for them to request the results.
           printf "%s - Waiting to send results...\n" "$(date +%T)"
           nc -lp ${commsPort} <${resultsFile}
           cat ${resultsFile}
        else
            printf "%s - Error encountered obtaining results from slave pods. Waiting to report failure status...\n" "$(date +%T)"
            nc -lp ${commsPort} <<<"FAILURE"
        fi
    else
        printf "%s - Unsupported mode: %s\n" "$(date +%T)" "${mode}"
        continue
    fi

    cd -
done
