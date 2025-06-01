#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017,2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
#
# This script can be used to execute a suite of cpu, memory and file io benchmarks to
# benchmark the performance of a server.
#
# The results are written to a file called sysbench_results_${host}.csv which contains
#  the results for all tests in one file.

function executeTest {
  DATE=$(date --utc +%FT%TZ)
  # echo "Running sysbench --test=${test_type} --threads=${num_threads} ${extra_args}"
  # File tests require a prepare/cleanup phase
  if [ "${setup_required}" = "true" ]
  then
    sysbench ${test_type} --threads=${num_threads} ${extra_args} prepare
  fi

  result=$(sysbench ${test_type} --threads=${num_threads} ${extra_args} run)
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
  echo "${DATE},${host},${machinetype},${worker},${test_type},${num_threads},${extra_args},${min},${avg},${max},${per95},${time},${events}" >> ${result_file}

  # File tests require a prepare/cleanup phase
  if [ "${setup_required}" = "true" ]
  then
    sysbench ${test_type} --threads=${num_threads} ${extra_args} cleanup
  fi
}

function getStat {
   line=$(echo "$1" | grep "$2")
   # Get last string on the line
   statistic=$(echo $line | rev | cut -d$" " -f1 | rev)
}

# main
# get hostname from the HOST_IP defined in the fieldPath: spec.nodeName in the config yaml
host="${HOST_IP}"
# map the machine to the machine type which has been passed into the pod as an env variable in the form nodex_x_x_x
hostenv="node$(echo ${host} | sed 's/\./_/g')"
machinemap="${hostenv}"
# split the mapped machine in two parts, the type and the worker number
# The ! in the next line is called variable indirect expansion.  See https://www.gnu.org/software/bash/manual/bashref.html#Shell-Parameter-Expansion
machinetype=$(echo ${!machinemap} | cut -d$"-" -f1)
worker=$(echo ${!machinemap} | cut -d$"-" -f2)

result_file=$1

# Write a header to the top
touch "${result_file}"
echo "Date,host,machinetype,worker,test_type,num_threads,extra_args,min,avg,max,95th percentile,total time,total events" > ${result_file}

  extra_args=""
  setup_required="false"
  now=$(date --utc +%FT%TZ)

  # run 1 of each test with 1 and 16 threads
  num_threads=1
  test_type=cpu
  # time=100
  executeTest

  num_threads=2
  test_type=cpu
  # time=100
  executeTest

  num_threads=4
  test_type=cpu
  # time=100
  executeTest

  num_threads=8
  test_type=cpu
  # time=100
  executeTest

  num_threads=1
  test_type=memory
  extra_args="--memory-block-size=1m"
  executeTest

  num_threads=2
  test_type=memory
  extra_args="--memory-block-size=1m"
  executeTest

  num_threads=4
  test_type=memory
  extra_args="--memory-block-size=1m"
  executeTest

  num_threads=8
  test_type=memory
  extra_args="--memory-block-size=1m"
  executeTest

  setup_required="true"
  extra_args="--file-test-mode=rndrw"
  num_threads=1
  test_type=fileio
  #time=100
  executeTest

  setup_required="true"
  extra_args="--file-test-mode=rndrw"
  num_threads=2
  test_type=fileio
  #time=100
  executeTest

  setup_required="true"
  extra_args="--file-test-mode=rndrw"
  num_threads=4
  test_type=fileio
  #time=100
  executeTest

  setup_required="true"
  extra_args="--file-test-mode=rndrw"
  num_threads=8
  test_type=fileio
  #time=100
  executeTest

  setup_required="false"
  extra_args=""

  cat $result_file
