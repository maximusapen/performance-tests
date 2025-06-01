#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017,2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# This script can be used to execute a suite of cpu, memory and file io benchmarks to
# benchmark the performance of a server.
#
# The results are written to a file called sysbench_results_${host}.csv which contains
#  the results for all tests in one file

function executeTest {
  DATE=$(date --utc +%FT%TZ)
  echo "Running sysbench --test=${test_type} --threads=${num_threads} ${extra_args}"
  # File tests require a prepare/cleanup phase
  if [ "${setup_required}" = "true" ]
  then
    sysbench --test=${test_type} --threads=${num_threads} ${extra_args} prepare
  fi

  result=$(sysbench --test=${test_type} --threads=${num_threads} ${extra_args} run)
  getStat "${result}" "min:"
  min=${statistic}
  getStat "${result}" "avg:"
  avg=${statistic}
  getStat "${result}" "max:"
  max=${statistic}
  getStat "${result}" "percentile:"
  per95=${statistic}
  getStat "${result}" "time:"
  time=${statistic}
  getStat "${result}" "events:"
  events=${statistic}
  echo "${DATE},${host},${test_type},${num_threads},${extra_args},${min},${avg},${max},${per95},${time},${events}" >> ${result_file}

  # File tests require a prepare/cleanup phase
  if [ "${setup_required}" = "true" ]
  then
    sysbench --test=${test_type} --threads=${num_threads} ${extra_args} cleanup
  fi
}

function getStat {
   line=$(echo "$1" | grep "$2")
   # Get last string on the line
   statistic=$(echo $line | rev | cut -d$" " -f1 | rev)
}
# main

# Set defaults
loops=1
interval=0
all=true
if [ $# != 0 ]
then
  loops=$1
  if [ -n "$2" ]
  then
    interval=$2
  fi
  if [ -n "$3" ]
  then
    all=$3
  fi
fi



# Make sure sysbench is installed (This will install latest version - default apt-get was quite old)
curl -s https://packagecloud.io/install/repositories/akopytov/sysbench/script.deb.sh | sudo bash
sudo apt -y install sysbench

host=$(hostname)
result_file="sysbench_results_${host}.csv"

# Keep Historical data in the `results file - but write a header to the top in case it isn't there
touch "${result_file}"
sed -i  '1iDate,host,test_type, num_threads,extra_args,min,avg,max,95th percentile,total time,total events' "${result_file}"

for (( i=1; i<=$loops; i++ )); do
  extra_args=""
  setup_required="false"
  now=$(date --utc +%FT%TZ)
  echo "$now Starting test $i of $loops"
  if [ $all == "true" ]
  then
    # We run each test twice to check repeatability, then run 1, 16 & 32 threads of each test
    # CPU Tests
    num_threads=1
    test_type=cpu
    executeTest

    num_threads=1
    test_type=cpu
    executeTest

    num_threads=16
    test_type=cpu
    executeTest

    num_threads=16
    test_type=cpu
    executeTest

    num_threads=32
    test_type=cpu
    executeTest

    num_threads=32
    test_type=cpu
    executeTest

    # Memory Tests

    num_threads=1
    test_type=memory
    executeTest

    num_threads=1
    test_type=memory
    executeTest

    num_threads=16
    test_type=memory
    executeTest

    num_threads=16
    test_type=memory
    executeTest

    num_threads=32
    test_type=memory
    executeTest

    num_threads=32
    test_type=memory
    executeTest

    # Disk Tests
    setup_required="true"
    extra_args="--file-test-mode=rndrw"

    num_threads=1
    test_type=fileio
    executeTest

    num_threads=1
    test_type=fileio
    executeTest

    num_threads=16
    test_type=fileio
    executeTest

    num_threads=16
    test_type=fileio
    executeTest

    num_threads=32
    test_type=fileio
    executeTest

    num_threads=32
    test_type=fileio
    executeTest
  else
    # Short version - just run 1 of each test
    num_threads=1
    test_type=cpu
    executeTest

    num_threads=1
    test_type=memory
    executeTest

    setup_required="true"
    extra_args="--file-test-mode=rndrw"
    num_threads=1
    test_type=fileio
    executeTest
  fi
  setup_required="false"
  extra_args=""

  echo "Sleeping for $interval seconds before next test"
  sleep $interval
done
