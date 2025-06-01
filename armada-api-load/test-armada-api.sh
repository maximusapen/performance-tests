#!/bin/bash

# Script to start Jmeter test.  Modify test config for your test below.

if [[ -z ${ARMADA_PERFORMANCE_API_KEY} ]]; then
    echo "Need to set armada_performance_api_key environment variable. Exiting"
    exit 1
fi

if [ $# -lt 4 ]; then
    echo "Usage:"
    echo "    test-armada-api.sh <carrier> <interval in sec> <thread number> <thread req limit in min> [<summary result file>]"
    echo "Example:"
    echo "  Run Jmeter for 10 min with 5 threads limiting each thread to 1200 req/min (20 req/sec) and write summary to default summary.out:"
    echo "    ./test-armada-api.sh origin-4 600 5 1200"
    echo "  Run Jmeter for 20 min with 2000 threads limiting each thread to 1200 req/min (20 req/sec) and write summary to summary_1000.out:"
    echo "    ./test-armada-api.sh origin-4 1200 2000 1200 summary_1000.out"
    echo "Warning: Do not set interval for longer than 1 hour or you will get RC-401,Unauthorized."
    exit 1
fi

carrier=$1     # carrier, e.g. origin-4 for carrier4
interval=$2    # he runlength in seconds for the test to run
thread=$3      # the number of JMeter threads to run
threadLimit=$4 # The throughput limit (!! requests/MINUTE !!) of each thread.
resCSVFile=$5  # Optional alternative file name to summary.out

declare -i testThroughput=${thread}*${threadLimit}/60

if [[ ${resCSVFile} == "" ]]; then
    resCSVFile="summary.out"
fi

echo "carrier: ${carrier}"
echo "interval: ${interval}"
echo "thread: ${thread}"
echo "threadLimit: ${threadLimit}"
echo "resCSVFile: ${resCSVFile}"
echo "testing throughput: ${testThroughput} req/sec"

resJTLFile="results_${thread}.jtl"
apiLogFile="armada-api-${thread}.log"

sudo rm ${resJTLFile}
sudo rm ${apiLogFile}
sudo rm ${resCSVFile}

printf "\n%s - Starting jmeter.\n" "$(date +%T)"
/usr/local/apache-jmeter/bin/jmeter -n -t armada-api.jmx -l ${resJTLFile} -j ${apiLogFile} -JRUNLENSEC=${interval} -JTHREAD=${thread} -JCARRIER=${carrier} -JTPUTMINS=${threadLimit} -JRAMPUPSECS=0
printf "\n%s - Starting JMeterPluginsCMD.\n" "$(date +%T)"
/usr/local/apache-jmeter/bin/JMeterPluginsCMD.sh --tool Reporter --plugin-type AggregateReport --input-jtl ${resJTLFile} --generate-csv ${resCSVFile}

# Check error
printf "\n%s - Checking errrors.\n" "$(date +%T)"
./checkRC.sh ${resJTLFile}
