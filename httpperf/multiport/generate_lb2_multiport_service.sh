#!/bin/bash

# Generate httpperf_lb2_multiport_service.yaml with 1000 ports starting at 30091

# release is http-perf or https-perf
release=$1
declare -i startPort=$2
declare -i endPort=$3

if [[ ${release} == "http-perf" ]]; then
    targetPort=8080
elif [[ ${release} == "https-perf" ]]; then
    targetPort=8443
else
    echo "Pass in release of http-perf or https-perf"
    exit 1
fi

echo Generating ${release} service with targetPort ${targetPort} for port range ${startPort} to ${endPort}

cat lb2_multiport_service_template.yaml | sed "s/RELEASE/${release}/g" >httpperf_lb2_multiport_service.yaml

for ((i = ${startPort}; i <= ${endPort}; i++)); do
    echo "    - name: http${i}" >>httpperf_lb2_multiport_service.yaml
    echo "      protocol: TCP" >>httpperf_lb2_multiport_service.yaml
    echo "      port: ${i}" >>httpperf_lb2_multiport_service.yaml
    echo "      targetPort: ${targetPort}" >>httpperf_lb2_multiport_service.yaml
done
chmod 775 httpperf_lb2_multiport_service.yaml
