#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019, 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Control over what is run
#testDuration=300
testDuration=600
enableInbound=true
enableOutbound=true
enableClusterSetup=true
runBaselineTest=false

# Number of times each test is run
testRuns=3
run200ReqPerSec=false
run800ReqPerSec=true

# Default to just 16 pods due to running this number on April private cluster code set.
serverPods=18
clientPods=16

# Comment out to drop chart generation
#enableCharts=" -c"

# Environment configuration
# NOTE: If you change this configuration you will likely have to setup the kube config under
# /performance/config and the files with the names defined by $privateKubeConfig,
# $remoteKubeConfig and $privateKubeConfig
# NOTE: See https://ibm.ent.box.com/notes/402133441251 for info on how to get this info
keystorePasscode=privclusterjuly23b

privateKubeConfig=privClusterJuly23b
privateSecret=privclusterjuly23b # pragma: allowlist secret
privateVlandID=2263903
privateIngress=privclusterjuly23b.dal09.stg.containers.appdomain.cloud

remoteKubeConfig=iperfClient
remoteSecret=iperfclient # pragma: allowlist secret
remoteVlandID=2263903
remoteIngress=iperfclient.dal09.stg.containers.appdomain.cloud

#remoteKubeConfig=iperfServer
#remoteSecret=iperfserver   # pragma: allowlist secret
#remoteVlandID=2263903
#remoteIngress=iperfserver.dal09.stg.containers.appdomain.cloud

if ${runBaselineTest}; then
    # Baseline test override for using iperfServer in place of privClusterJuly23b.
    # Set to non-private cluster
    privateKubeConfig=iperfServer
    privateSecret=iperfserver # pragma: allowlist secret
    privateVlandID=2263903
    privateIngress=iperfserver.dal09.stg.containers.appdomain.cloud
fi

environment=stage5
# End: Environment configuration

testPlansDir=/performance/armada-perf/jmeter-dist/tests
testPlanDir=${testPlansDir}/inbound.plan

if ${runBaselineTest}; then
    # It must have a space at the end
    baselineTxt="baseline "
else
    baselineTxt=""
fi

function deployHttpperfPods() {
    echo "${testName}: Deploying httpperf to $KUBECONFIG"
    pushd /performance/armada-perf/httpperf/imageDeploy
    helm install httpperf httpperf --namespace httpperf --set image.name=armada_performance_${environment}/httpperf,replicaCount=${serverPods},ingress.hosts={${INGRESS}},ingress.tls.hosts={${INGRESS}},ingress.tls.secretName=${SECRET},service.loadBalancer2.zone=dal09,service.loadBalancer2.vlanID=\"${VPN}\"
    sleep 60
    popd
}

function setupJMeterForLB2() {
    clusterDef=$(kubectl -n httpperf get svc | grep httpperf-lb2-service | awk '{print "ingress,http,"$4","$5}' | cut -d: -f1)
    if [[ -z ${clusterDef} ]]; then
        echo "${testName}: ERROR: Couldn't get httpperf-lb2-service"
        exit 1
    fi

    if [[ ! -d ${testPlanDir} ]]; then
        mkdir -p ${testPlanDir}
        cp ${testPlansDir}/httpperf/test.jmx ${testPlanDir}
        for ((i = 1; i <= 1000; i++)); do
            echo "GET,/request?size=1500," >>${testPlanDir}/requests.csv
        done
    fi
    echo "${testName}: Populating clusters.csv with ${clusterDef}"
    rm -f ${testPlanDir}/clusters.csv
    for ((i = 0; i < 1000; i++)); do
        echo ${clusterDef} >>${testPlanDir}/clusters.csv
    done

    if [[ ! -f ${testPlanDir}/cert.jks ]]; then
        echo "You must generate a keyfile in ${testPlanDir} to run this test. Using defaults is fine."
        echo "Ex: keytool -genkey -alias ${privateIngress} -keyalg RSA -keystore cert.jks -keysize 2048 -storepass ${keystorePasscode}"
        exit 1
    fi

    kubectl -n httpperf get pods -o wide
}

function setupJMeterForNodePorts() {
    nodes=$(kubectl get nodes -o jsonpath='{range .items[*]}{.status.addresses[?(@.type=="ExternalIP")].address}{"\n"}' | grep "^169")
    port=$(kubectl get service -n httpperf httpperf-np-service -o jsonpath='{.spec.ports[0].nodePort}')
    while (($count < 1000)); do
        for i in $(echo $nodes); do
            echo "ingress,http,${i},${port}" >>tests/inbound.plan/clusters.csv
            count=$((count + 1))
            if [[ $count -ge 1000 ]]; then
                break
            fi
        done
    done
}

function deployJmeterPods() {
    echo "${testName}: Deploying JMeter to $KUBECONFIG"
    pushd /performance/armada-perf/jmeter-dist
    ./bin/install.sh -t 4 -r 20000 -d 30 -n httpperf -s ${clientPods} -p ${keystorePasscode} -e ${environment} -m standalone inbound.plan
    sleep 60
    popd

    kubectl -n httpperf get pods -o wide
}

function removeCharts() {
    helm uninstall httpperf --namespace httpperf
    helm uninstall jmeter-standalone --namespace httpperf
    helm uninstall jmeter-dist --namespace httpperf
    sleep 60
}

function removeHttpperfCharts() {
    helm uninstall httpperf --namespace httpperf
    sleep 60
}

function removeJmeterCharts() {
    helm uninstall jmeter-standalone --namespace httpperf
    helm uninstall jmeter-dist --namespace httpperf
    sleep 60
}

function setupNamespace() {
    kubectl get ns | grep httpperf >/dev/null 2>&1
    if [[ $? -ne 0 ]]; then
        echo "Creating httpperf namespace in $KUBECONFIG"
        /performance/armada-perf/automation/bin/setupRegistryAccess.sh httpperf
    fi
}

function runTest() {
    threads=$1
    rate=$2
    tag=$3
    enableCs=$4
    echo "Starting test: You aren't likely to see any output until it is done"
    ./bin/run.sh -t ${threads} -r ${rate} -d ${testDuration} -n httpperf ${enableCs} | sed -ue "s/^TOTAL\(.*\)$/${tag},TOTAL\1,$(($rate * 60))/g"
}

function runBatchOfTests() {
    direction=$1

    pushd /performance/armada-perf/jmeter-dist
    test="${direction} ${baselineTxt}${testName}"
    if ${run200ReqPerSec}; then
        echo "$(date +%Y-%m-%d_%T) ${test}: -t 3125 -r 200 -d ${testDuration}"
        for ((i = 1; i <= ${testRuns}; i++)); do
            if [[ $i -eq ${testRuns} ]]; then
                #./bin/run.sh -t 3125 -r 200 -d ${testDuration} -n httpperf ${enableCharts}
                runTest 3125 200 "${test}" ${enableCharts}
            else
                #./bin/run.sh -t 3125 -r 200 -d ${testDuration} -n httpperf
                runTest 3125 200 "${test}"
            fi
        done
    fi

    if ${run800ReqPerSec}; then
        echo "$(date +%Y-%m-%d_%T) ${test}: -t 750 -r 800 -d ${testDuration}"
        for ((i = 1; i <= ${testRuns}; i++)); do
            # Only enable jmeter charts on the last run
            if [[ $i -eq ${testRuns} ]]; then
                #./bin/run.sh -t 750 -r 800 -d ${testDuration} -n httpperf ${enableCharts}
                runTest 750 800 "${test}" ${enableCharts}
            else
                #./bin/run.sh -t 750 -r 800 -d ${testDuration} -n httpperf
                runTest 750 800 "${test}"
            fi
        done
    fi
    popd
    if [[ -n ${enableCharts} ]]; then
        mv /performance/armada-perf/jmeter-dist/charts.20*.tar.gz .
    fi
}

if [[ $# -le 0 ]]; then
    echo "${testName}: ERROR: Must name the test being run. Don't use ','. Should be of the form '<machine type>[ <qualifier2>]'. Ex: ./privateClusterTest.sh c2c.16x16 rel:6-24"
    exit 1
fi

testName="$@"
if ${enableInbound}; then
    if ${enableClusterSetup}; then
        echo "$(date +%Y-%m-%d_%T) ${testName}: Deploying inbound to ${privateKubeConfig} from ${remoteKubeConfig} test"
        . ${privateKubeConfig}
        setupNamespace
        removeHttpperfCharts
        VPN=${privateVlandID}
        INGRESS=${privateIngress}
        SECRET=${privateSecret}
        deployHttpperfPods
        setupJMeterForLB2

        . ${remoteKubeConfig}
        setupNamespace
        removeJmeterCharts
        deployJmeterPods
    else
        . ${remoteKubeConfig}
    fi

    runBatchOfTests inbound
fi

if ${enableOutbound}; then

    if ${enableClusterSetup}; then
        echo "$(date +%Y-%m-%d_%T) ${testName}: Deploying outbound from ${privateKubeConfig} to ${remoteKubeConfig} test"
        . ${remoteKubeConfig}
        setupNamespace
        removeHttpperfCharts
        # iperfServer2 vlan
        VPN=${remoteVlandID}
        INGRESS=${remoteIngress}
        SECRET=${remoteSecret}
        deployHttpperfPods
        setupJMeterForLB2
        #setupJMeterForNodePorts

        . ${privateKubeConfig}
        setupNamespace
        removeJmeterCharts
        deployJmeterPods
    else
        . ${privateKubeConfig}
    fi

    runBatchOfTests outbound
fi

echo "$(date +%Y-%m-%d_%T) ${testName}: Done with tests"
