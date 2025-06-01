#!/bin/bash

# By default run in server mode - but if the -c argument is passed, then just use the arguments supplied
client_mode=false
for var in "$@"
do
    if [[ $var == "-c" ]]; then
      client_mode=true
    fi
done

if [[ $client_mode == true ]]; then
    args=${*}
else
    args="-s ${*}"
fi

echo "args = $args"
iperf3 $args
