#!/bin/bash -e

waitForDNSUpdate() {
    location=$1
    maxWaitTime=$2

    dnsUpdated=false
    curWaitTime=0
    pollingInterval=60

    while [[ ${curWaitTime} -lt ${maxWaitTime} ]]; do
        dnsrec=$(${perf_dir}/bin/armada-perf-client2 sat location dns ls --location ${location} --json | jq -r '.[0]')
        readarray -t dnsips < <(echo ${dnsrec} | jq -r '.nlbIPArray | .[]')
        dnsname=$(echo $dnsrec | jq -r '.nlbHost')
        resolvedIP=$(dig ${dnsname} +short | tail -n1)
        for ip in "${dnsips[@]}"; do
            if [[ ${ip} == ${resolvedIP} ]]; then
                dnsUpdated=true
                break 2
            fi
        done

        printf "%s - Waiting for '%s' DNS records to be updated.\n" "$(date +%T)" "${location}"

        sleep ${pollingInterval}
        ((curWaitTime += ${pollingInterval}))
    done

    if [[ "${dnsUpdated}" != true ]]; then
        printf "\n%s - Gave up waiting for '%s' DNS records to be updated. Exiting.\n\n" "$(date +%T)" "${location}"
        exit 1
    fi
}

perf_dir=/performance
armada_perf_dir=${perf_dir}/armada-perf

location=$1

# Get all control plane hosts
locationHosts=$(${perf_dir}/bin/armada-perf-client2 sat host ls --location ${location} --json)
controlPlaneHosts=$(echo ${locationHosts} | jq -r '.[] | select(.assignment.clusterName=="infrastructure") | .name')
readarray -t controlPlaneHostsArr <<<${controlPlaneHosts}

# Check for multizone Satellite control plane
zoneCount=$(echo ${locationHosts} | jq -n '[inputs[] | select(.assignment.clusterName=="infrastructure") | .assignment.zone] | unique | length')
multizone=$((zoneCount > 1))

declare -A zones
IPs=()

# For each Satellite control plane host
for satelliteHost in "${controlPlaneHostsArr[@]}"; do
    # For multizone clusters, we need one IP from each zone
    if ((multizone)); then
        hostZone=$(echo ${locationHosts} | jq --arg SATHOST "${satelliteHost}" -r '.[] | select(.name==$SATHOST) | .assignment.zone')

        if [[ -v "zones[${hostZone}]" ]]; then
            continue
        else
            zones[${hostZone}]=""
        fi
    fi

    # Get the floating IP for this host
    FIP=$(${perf_dir}/bin/tomlToJson ${armada_perf_dir}/armada-perf-client2/config/perf-infrastructure.toml | jq --arg SATHOST "${satelliteHost}" --arg LOCATION "${location}" -r '.satellite.location | .[$LOCATION].hosts.control.servers | .[$SATHOST].vpc.floating_ip')
    IPs+=($FIP)

    # Limit to 3 IPs as that's what the dns register command requires
    if ((${#IPs[@]} == 3)); then
        break
    fi
done

printf "\n%s - Registering public IPs with the location's DNS\n" "$(date +%T)"
for ip in "${IPs[@]}"; do
    printf "\t-%s\n" ${ip}
    registerIPs=${registerIPs}"--ip ${ip} "
done

# Register the public IPs
${perf_dir}/bin/armada-perf-client2 sat location dns register --location ${location} ${registerIPs}

# Can take a while for DNS registration to complete, wait for up to 15 minutes
waitForDNSUpdate ${location} 900
