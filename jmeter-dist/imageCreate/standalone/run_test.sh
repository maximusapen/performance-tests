#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019, 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

commsPort=4444

while true; do
    # Wait for a test command to be sent by the user via the standalone pod's port
    printf "%s - Waiting for test command....\n" "$(date +%T)"

    # We'll get the client's ip for audit purposes
    cmd=$( nc -lnvp ${commsPort} 2> connection)
    clientIP=$(sed -nr "s/.*from.*\[([0-9\.]*)].*/\1/p" connection)

    IFS=' ' read -r -a cmd_data <<<"${cmd}"
    case "${cmd_data[0]}" in
    "START")
        printf "%s - \"%s\" received from %s\n" "$(date +%T)" "${cmd}" "${clientIP}"
        if (( ${#cmd_data[@]} > 1)); then
            # Override the number of JMeter threads (if specified)
            param="${cmd_data[1]}"
            JMETER_ARGS=$( echo ${JMETER_ARGS} | sed "s/-JTHREAD=[0-9]* //")
            JMETER_ARGS="${JMETER_ARGS} -JTHREAD=${param} "
        fi
        if (( ${#cmd_data[@]} > 2)); then
            # Override the test throughput (if specified)
            param=$(("${cmd_data[2]}" * 60 ))
            JMETER_ARGS=$( echo ${JMETER_ARGS} | sed "s/-JTPUTMINS=[0-9]* //")
            JMETER_ARGS="${JMETER_ARGS} -JTPUTMINS=${param} "
        fi
        if (( ${#cmd_data[@]} > 3)); then
            # Override the test duration (if specified)
            param="${cmd_data[3]}"
            JMETER_ARGS=$( echo ${JMETER_ARGS} | sed "s/-JRUNLENSEC=[0-9]* //")
            JMETER_ARGS="${JMETER_ARGS} -JRUNLENSEC=${param} "
            #olb-java tests uses different param for run length
            JMETER_ARGS=$(echo ${JMETER_ARGS} | sed "s/-GDURATION=[0-9]* //")
            JMETER_ARGS="${JMETER_ARGS} -GDURATION=${param} "
        fi
        ;;
    "STOP" | "EXIT")
        printf "%s - \"%s\" received from %s\n" "$(date +%T)" "${cmd}" "${clientIP}"
        break 2
        ;;
    *)
        printf "%s - Unrecognised command \"%s\" received. Ignoring.\n" "$(date +%T)" "${cmd}"
        continue
        ;;
    esac

    printf "%s - Starting test.\n" "$(date +%T)"
    if [[ -d "results" ]]; then
        rm -r results
    fi

    mkdir -p results
    cd results

    # Dummy file for request body contents
    touch .json

    # Start tests
    set -x
    jmeter -Dserver.rmi.ssl.disable=true -Dhttpclient4.time_to_live=3600000 -n -t ../test.jmx -f -l samples.out ${JMETER_ARGS}
    set +x

    # samples.out is pulled by master
    head samples.out

    cd -
done

