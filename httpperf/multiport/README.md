# Multiport for httpperf

## Changes

Currently, changes in runAuto.sh is documented as this was an ad-hoc testing.
If we do need to run this more often, then the following changes should be
included in runAuto.sh with test name like `http-perf-multiport`, `https-perf-multiport`.
The number of ports can also be reduced/configurable if included in daily runAuto.sh testing.

### Change 1

In httperfJMeterRun, add code to extract jmeter master logs.  This allows you to check the distributed loader startup and termination.
You may have to ignore those error results when jmeter starts terminating test threads.

            "${jmeter_dist_dir}/bin/run.sh" -t ${numThreads} -r ${threadTPutRPS} -d 300 >"${resCSVFile}"
            cat ${resCSVFile}

    ###  Begin Add Block  ###
            echo "Getting jmeter master pod logs - $(pwd)/${protocol}${testType}-${numThreads}-*"
            jmeter_master_pod=$(kubectl get pod | grep jmeter-dist-master | awk '{print $1}')

            kubectl cp default/${jmeter_master_pod}:/jmeter/results/jmeter-master.log ./${protocol}${testType}-${numThreads}-jmeter-master.log >/dev/null
            kubectl cp default/${jmeter_master_pod}:/jmeter/results/jmeter.log ./${protocol}${testType}-${numThreads}-jmeter.log >/dev/null
            kubectl cp default/${jmeter_master_pod}:/jmeter/results/results.csv ./${protocol}${testType}-${numThreads}-results.csv >/dev/null

            echo "This may take a while for samples.out"
            kubectl exec -it -n default ${jmeter_master_pod} tar cvfz results/samples.tgz results/samples.out
            kubectl cp default/${jmeter_master_pod}:/jmeter/results/samples.tgz ./${protocol}${testType}-${numThreads}-samples.tgz >/dev/null

            echo
            echo ${protocol}${testType}-${numThreads}-jmeter-master.log
            echo "****************************************************"
            cat ${protocol}${testType}-${numThreads}-jmeter-master.log
            echo "****************************************************"
            echo
    ###  End Add Block  ###

### Change 2

In generateHTTPPerfCSV, add code block for endPort:

    port=$4              <---  delete

    numCSVRepeats=200    <---  delete

    ###  Begin Add Block  ###

    declare -i port=$4
    declare -i endPort=$4

    if [ -n "$5" ]; then
        endPort=$5
    fi
    numCSVRepeats=2000

    ###  End Add Block  ###

    printf "\n%s - Generating JMeter hosts for %s %s %s testing\n\n" "$(date +%T)" "${mode}" "${protocol}" "${test}"

### Change 3

In generateHTTPPerfCSV for loadbalancer2, replace with this block using httpperf-lb2-multiport-service:

    "loadbalancer2")
        target_ip=$(kubectl describe service httpperf-lb2-multiport-service -n "${perftest}" | grep "LoadBalancer Ingress" | awk '{print $3}')
        startPort=${port}
        for ((i = 1; i <= $numCSVRepeats; i++)); do
            if [[ ${mode} == "standalone" ]]; then
                echo $target_ip >>${hostsCSVFile}
            else
                echo "node1,${protocol},${target_ip},${port}" >>${hostsCSVFile}
                port=${port}+1
                if [[ ${port} > ${endPort} ]]; then
                    port=${startPort}
                fi
            fi
        done
        echo "Quick check on ${hostsCSVFile}"
        tail -10 ${hostsCSVFile}
        ;;

### Change 4

To check all ports are available before starting test, add new waitForConnection() to runAuto.sh.

waitForConnection() {
    < Copy from httpperf/multiport/testport.sh>
}

### Change 5

In tests "http-perf" | "https-perf", add code blocks for deploy steps specifying the number of ports to test in nPort:

            # Now deploy the application...
            printf "\n%s - Installing ${imageName} application on ${run_on_cluster}\n" "$(date +%T)"
            set -x
            ${helm_dir}/helm install "${perftest}" "${http_perf_dir}/imageDeploy/${imageName}" --namespace "${perftest}" --set metricsPrefix="${METRICS_PREFIX}" --set k8sVersion="${K8S_SERVER_VERSION}" --set image.name="armada_performance_${ns_suffix}/${imageName}",replicaCount="${replicas}${httpZones}${ingress_deployment}"${loadBalancer2}
            set +x

            ###  Begin Change Block  ###
            declare -i startPort=30091
            declare -i nPort=100    # Change for testing
            declare -i endPort=${startPort}+${nPort}-1
            printf "\n%s - Generating multiport/httpperf_lb2_multiport_service.yaml with nPort:${nPort} from port(s) ${startPort} to ${endPort}\n" "$(date +%T)"
            cd multiport
            ./generate_lb2_multiport_service.sh ${perftest} ${startPort} ${endPort}
            cd -
            printf "\n%s - Creating httpperf_lb2_multiport_service\n" "$(date +%T)"
            kubectl create -f multiport/httpperf_lb2_multiport_service.yaml -n ${perftest}
            printf "\n%s - Created httpperf_lb2_multiport_service\n" "$(date +%T)"
            sleep 60
            kubectl get service -n ${perftest}
            host=$(kubectl describe service httpperf-lb2-multiport-service -n "${perftest}" | grep "LoadBalancer Ingress" | awk '{print $3}')
            for ((i = ${startPort}; i <= ${endPort}; i++)); do
                waitForConnection ${host} ${i}
            done
            ###  End Change Block  ###

            # Wait to ensure application is fully up-and-running and listening for requests
            printf "\n%s - Waiting 3 mins to ensure \"${perftest}\" application is started\n" "$(date +%T)"
            sleep 180

### Change 6

In tests "http-perf" | "https-perf", replace code to call modified generateHTTPPerfCSV with start and end ports.
Also increase the duration of distributed threads to run from 5 minto 20 min with "-d 1200" in ${jmeter_dist_dir}/bin/install.sh.

When testing with a large cluster, say, 100 nodes with many ports, it will take some time for jmeter to startup and terminate.
Running for 5 min is too short and will results in many errors.

                # Classic Load Balancers
                if [[ "${cluster_type}" == "classic" ]]; then

                    ###  Begin Change Block  ###
                    # Load Balancer 2
                    #testPort=${testPorts["${protocol}-loadbalancer2"]}
                    #generateHTTPPerfCSV loadbalancer2 distributed ${protocol} ${testPort}
                    testPort=30091
                    echo Generating ${protocol} for ports ${startPort} to ${endPort}
                    generateHTTPPerfCSV loadbalancer2 distributed ${protocol} ${startPort} ${endPort}
                    export KUBECONFIG=${load_kubeconfig}
                    ${jmeter_dist_dir}/bin/install.sh -t 5 -r 250 -d 1200 -s 30 -e "${ns_suffix}" "${imageName}"

                    ###  End Change Block  ###

                    sleep 30

                    export KUBECONFIG=${load_kubeconfig}

### Change 7

Change the duration of jmeter test from 5 min to match the duration of distributed loader test threads modified in Change 6 above.

In httperfJMeterRun(), change "-d 300" to "-d 1200" for 20 min test run:

            ###  Begin Change Block  ###
            "${jmeter_dist_dir}/bin/run.sh" -t ${numThreads} -r ${threadTPutRPS} -d 1200 >"${resCSVFile}"
            ###  End Change Block  ###

### Change 8

Remove the other tests in http(s)-perf that you are not interested to save testing time.

### Change 9

If ingress is causing issue, remove call to waitForIngress as lb2 test do not require ingress.

If helm install is failing due to ingress, add "--set ingress.enabled=false" to helm install of http-perf

## Testing

Back up /performance/src/github.ibm.com/alchemy-containers/armada-performance/automation/bin/runAuto.sh on client to be reverted back after your test.  Replace script with your modified runAuto.sh.

You have option to modify the followings in runAuto.sh for your test:

- duration
  - To change duration from 20 min.  See change 6 and change 7.
- number of ports
  - See change 5 with following line.  Change nPort and run tests with different ports, say, 1, 100, 200, 500.
    - declare -i nPort=100    # Change for testing

### Time for ports to be available

A simple testport.sh is provided to measure how long it takes for all 1000 ports to be available if there is concern about the time to take ports to be available

Steps:

- ./generate_lb2_multiport_service.sh [ http | https ] < startPort> <{endPort>, e.g. ./generate_lb2_multiport_service.sh https 30091 31090
- kubectl create -f multiport/httpperf_lb2_multiport_service.yaml -n [ http | https ]
- ./testport.sh

You can check the service to verify the many ports created at any time with:

- kubectl get service -n [ http-perf | https-perf ]

### Errors after remote hosts terminated

You may want to ignore the errors and metrics after remote hosts started to terminate.  Example below:

2021-11-02 12:57:00,001 INFO o.a.j.r.Summariser: summary + 350000 in 00:00:30 = 11667.8/s Avg:    25 Min:     0 Max:   341 Err:     0 (0.00%) Active: 300 Started: 50 Finished: 0

2021-11-02 12:57:00,002 INFO o.a.j.r.Summariser: summary = 13684789 in 00:19:36 = 11641.1/s Avg:    25 Min:     0 Max: 60829 Err:   982 (0.01%) <--- use this metrics for results

2021-11-02 12:57:24,882 INFO o.a.j.JMeter: Finished remote host: 172.30.53.160 (1635857844882)
2021-11-02 12:57:24,945 INFO o.a.j.JMeter: Finished remote host: 172.30.251.215 (1635857844945)
::::::::
2021-11-02 12:57:56,803 INFO o.a.j.JMeter: Finished remote host: 172.30.68.152 (1635857876803)
2021-11-02 12:57:57,110 INFO o.a.j.JMeter: Finished remote host: 172.30.53.158 (1635857877109)
2021-11-02 12:57:57,749 INFO o.a.j.r.Summariser: summary +   1232 in 00:00:27 =   46.1/s Avg:  1562 Min:     1 Max: 31596 Err:     0 (0.00%) Active: 0 Started: 50 Finished: 300

2021-11-02 12:57:57,750 INFO o.a.j.r.Summariser: summary = 13993597 in 00:20:33 = 11346.4/s Avg:    25 Min:     0 Max: 60829 Err:   982 (0.01%) <--- instead of this

2021-11-02 12:57:57,751 INFO o.a.j.JMeter: Finished remote host: 172.30.179.224 (1635857877751)


### Investigation

- lb_multiport_service_template.yaml and generate_lb_multiport_service.sh are provided for investigation
comparing lb and lb2 behaviour only.
- Check ibm-cloud-provider-ip logs in ibm-system namespace.  If 2 edge nodes are used in these, the ibm-cloud-provider-ip pods runs on the 2 edge nodes only with one active edge node and the other acting as backup
- Check ipvs on active edge node with runon (https://github.ibm.com/kubernetes-tools/runon).

  - runon < node > command apt install ipvsadm
  - runon < node > command ipvsadm -Ln --stats > ipvsadm.log

    - Check ipvsadm.log.  There should be traffic shown for each of the node (except for the edge nodes) and each of the ports.
