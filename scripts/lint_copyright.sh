#!/usr/bin/env bash
#*******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

function print_fmt_msg {
    echo "============== $(date -u +'%D %T %Z') =================="
    echo -e "  $1"
    echo "======================================================="
}

# Find all .go, .py and .sh files
FILES=($(find . -type f -name '*.sh' -not -path "./k8s-netperf/*" -not -path "./http-scale/*" -not -path "./vendor/*" -not -path "./k8s.io/*" -not -path "./test-fixtures/*" -o -name '*.py' -not -path "./k8s-netperf/*" -not -path "./http-scale/*" -not -path "./vendor/*" -not -path "./test-fixtures/*" -o -name '*.go' -not -path "./k8s-netperf/*" -not -path "./http-scale/*" -not -path "./vendor/*" -not -path "./k8s.io/*" -not -path "./kubernetes/perf-tests/*" -not -path "./test-fixtures/*"))
BAD_FILES=()
for FILE in "${FILES[@]}"; do
cat ${FILE} | grep 'IBM Cloud Kubernetes Service, 5737-D43'
return_code=$?
	if [[ ${return_code} != 0 ]]; then
	BAD_FILES+=(${FILE})
	fi
done

if [[ ${#BAD_FILES[@]} != 0 ]]; then
	print_fmt_msg "Fail Copyright Test"
	echo "Add the copyright to these files: ${BAD_FILES[*]}"
        exit 1
else
	print_fmt_msg "Success :) Passed Copyright Test"
        exit 0
fi
