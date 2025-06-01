#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

capture_date=$1

rm -rf backup/${capture_date}.setup.errors backup/${capture_date}.test.errors backup/${capture_date}.test.summary.txt backup/${capture_date}.test.dates
for log_file in `ls backup/*${capture_date}.log`; do 
    start_line=$(awk '/Created test end watch for/{ print NR; exit }' ${log_file})
    # TODO error processing doesn't work on lease log files, since they don't have "Created test end watch for" message
    if [[ -n ${start_line} ]]; then
        #head -${start_line} ${log_file} | tail -1
        head -${start_line} ${log_file} | grep error >> backup/${capture_date}.setup.errors
        tail -n +${start_line} ${log_file} | egrep "Client [0-9]* operation" >> backup/${capture_date}.test.errors
        tail -n +${start_line} ${log_file} | head -1 | cut -d: -f1-2 >> backup/${capture_date}.test.dates
        tail -1 ${log_file} | cut -d: -f1-2 >> backup/${capture_date}.test.dates
    fi
done

start_time=$(sort backup/${capture_date}.test.dates | head -1)
end_time=$(sort backup/${capture_date}.test.dates | tail -1)
echo "Test range: ${start_time} ${end_time}" >> backup/${capture_date}.test.summary.txt
echo >> backup/${capture_date}.test.summary.txt

echo "Total test errors: $(wc -l backup/${capture_date}.test.errors | awk '{print $1}')" >> backup/${capture_date}.test.summary.txt
echo >> backup/${capture_date}.test.summary.txt

echo "Test error counts" >> backup/${capture_date}.test.summary.txt
egrep "Client [0-9]* operation"  backup/${capture_date}.test.errors | sed -e "s/^.*error://g" -e "s/ [0-9]* microseconds/ <ms> microseconds/g" | sort | uniq -c >> backup/${capture_date}.test.summary.txt
echo >> backup/${capture_date}.test.summary.txt

echo "Test error timeline" >> backup/${capture_date}.test.summary.txt
cut -d: -f1-3 backup/${capture_date}.test.errors | sed -e "s/\.[0-9]* .*$//g" | sort | uniq -c >> backup/${capture_date}.test.summary.txt
echo >> backup/${capture_date}.test.summary.txt

echo "Total setup errors: $(wc -l backup/${capture_date}.setup.errors | awk '{print $1}')" >> backup/${capture_date}.test.summary.txt
echo >> backup/${capture_date}.test.summary.txt

echo "Setup error counts" >> backup/${capture_date}.test.summary.txt
egrep "Client [0-9]* operation"  backup/${capture_date}.setup.errors | sed -e "s/^.*error://g" -e "s/ [0-9]* microseconds/ <ms> microseconds/g" | sort | uniq -c >> backup/${capture_date}.test.summary.txt

echo "Error data: backup/${capture_date}.test.summary.txt"
