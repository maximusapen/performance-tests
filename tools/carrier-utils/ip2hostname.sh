#!/bin/bash

# Script to convert carrier IP addresses to host names, from ether stdin
# of a file given as a parameter.
# It requires access to either calicoctl (so run on the master or workers)
# or a /etc/hosts file containing the carrier worker hosts


OIFS=$IFS
IFS=$'\n'

hostnames=""
convert_cmd="sed "
if [ -f "/opt/bin/calicoctl" ]; then
   convert_cmd="sed $(/opt/bin/calicoctl get nodes -o wide | grep -v NAME | sed -e '/^\s*$/d' -e "s/\/[0-9]*//g" -e "s/\.[[:alnum:].]* *(.*)//g" | awk '{ printf "-e \"s/%s/%s/g\"\n", $2, $1}' | sort | awk '{ printf "%s ", $0}')"
else
   hostnames=$(cat /etc/hosts)
   convert_cmd="sed $(grep "^10" /etc/hosts | sort | egrep "stage|prod|dev" | awk '{ printf "-e \"s/%s/%s/g\" ", $1, $2}')"
fi

eval $convert_cmd < "${1:-/dev/stdin}"
