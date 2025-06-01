#!/bin/bash

if [ $# -lt 1 ]; then
    echo "Usage:"
    echo "    checkRC.sh <result_jtl_file>"
    exit 1
fi
resJTLFile=$1
errorFile="errors_${resJTLFile}"
errorTmpFile="tmp_${errorFile}"
sudo rm ${errorFile}

# Script to check total number of Jmeter errors in results jtl file
grep -v "200,OK" ${resJTLFile} | grep -v timeStamp >${errorFile}
echo "Total number of errors extracted to ${errorFile}"
wc -l ${errorFile}

# When adding httpErrors, replace space " " with underscore "_"
declare -a httpErrors=("504,Gateway_Timeout" "503,Service_Unavailable" "500,Internal_Server_Error" "429,Too_Many_Requests" "401,Unauthorized" "404,Not_Found" "403,Forbidden" "400,Bad_Request" "Connection_reset" "Read_timed_out" "NoHttpResponseException" "UnknownHostException" "Cannot_assign_requested_address" "ConnectTimeoutException")

cp ${errorFile} /tmp/${errorFile}

for httpError in ${httpErrors[@]}; do
    # Print out the number of errors for the error type
    error="$(echo ${httpError} | sed "s/_/ /g")"
    numErrors=$(grep "${error}" ${resJTLFile} | wc -l)
    echo "Number of ${error}: ${numErrors}"
    # Save a list of other remaining errors on list
    errorsToCheck=$(echo ${errorsToCheck} | grep -n -v "${error}")
    grep -v "${error}" /tmp/${errorFile} >/tmp/${errorTmpFile}
    cp /tmp/${errorTmpFile} /tmp/${errorFile}
done

# Check for other errors
echo "Other errors found (may include last incomplete line in ${resJTLFile} which you can ignore):"
cat /tmp/${errorFile}
rm /tmp/${errorTmpFile} /tmp/${errorFile}
echo
echo "You can add other errors to httpErrors in checkRC.sh to be counted for future runs."
echo
