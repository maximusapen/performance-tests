#!/bin/bash

# Generate httpperf_lb_multiport_service.yaml with 1000 ports starting at 30081

# release is http-perf or https-perf
release=$1

if [[ ${release} == "http-perf" ]]; then
    targetPort=8080
elif [[ ${release} == "https-perf" ]]; then
    targetPort=8443
else
    echo "Pass in release of http-perf or https-perf"
    exit 1
fi

echo Generating ${release} service with targetPort ${targetPort}

cat lb_multiport_service_template.yaml | sed "s/RELEASE/${release}/g" >httpperf_lb_multiport_service.yaml
declare -i port=30081
for i in {1..1000}; do
    echo "    - name: http${i}" >>httpperf_lb_multiport_service.yaml
    echo "      protocol: TCP" >>httpperf_lb_multiport_service.yaml
    echo "      port: ${port}" >>httpperf_lb_multiport_service.yaml
    echo "      targetPort: ${targetPort}" >>httpperf_lb_multiport_service.yaml
    port=${port}+1
done
chmod 775 httpperf_lb_multiport_service.yaml
