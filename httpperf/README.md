# Armada httpperf Scalability Tests

These regression tests stress the performance of NodePorts, Loadbalancers and Ingress. The tests use a JMeter client to send http and https requests to a GO HTTP server pod running in a kubernetes cluster.

## Setting up the Client Machine and Building the Images

- Install the required prerequisite programs using the  [install-multiple-prereqs-on-perf-client](https://alchemy-containers-jenkins.swg-devops.com/job/Armada-Performance/job/perfClient-setup/job/install-multiple-prereqs-on-perf-client/) Jenkins job.
- Build and install the automation test code using the [Build-and-Copy-repo](https://alchemy-containers-jenkins.swg-devops.com/job/Armada-Performance/job/perfClient-setup/job/Build-and-Copy-perf-repo/) Jenkins Job.
- Build the test images using the [Publish-Perf-Docker-Images](https://alchemy-containers-jenkins.swg-devops.com/job/Armada-Performance/job/perfClient-setup/job/Publish-Perf-Docker-Images/)

## Running the Tests in the Automation

- The tests can be run within the automation framework using the [Run-Performance-Tests](https://alchemy-testing-jenkins.swg-devops.com/view/Armada-performance/job/Armada-Performance/job/Automation/job/Run-Performance-Tests/) Jenkins job. Select httpperf as the PERF_TEST.

## Deploying the httpperf Tests Manually

The tests can also be deployed and run manually as follows (this assumes the client has been setup, a cluster created and the httpperf images have been deployed using the Jenkins jobs mentioned above)

- On the client machine
  - Setup KUBECONFIG for your cluster (substitute YOUR_CLUSTER_NAME below)
    - cd to /performance/config dir on the client
    - Get the cluster config files : /performance/bin/armada-perf-client -action=GetClusterConfig -admin -clusterName=YOUR_CLUSTER_NAME
    - Unzip the downloaded kubeconfig file
    - Rename the resultant directory to YOUR_CLUSTER_NAME
      - It is good practice to change the permissions on this directory in case you want to run the automation against the cluster later:
        - sudo chmod -R 775 YOUR_CLUSTER_NAME
        - sudo chown jenkins -R YOUR_CLUSTER_NAME
    - Edit the yaml file in that directory so the certificates have absolute paths i.e. add /performance/config/YOUR_CLUSTER_NAME/ in front of each of the following pem files:
      - certificate-authority
      - client-certificate
      - client-key
    - Export kubeconfig:
    - export KUBECONFIG=/performance/config/YOUR_CLUSTER_NAME/YAML_FILENAME
    - Create the http namespace and add the registry secret

      - /performance/armada-perf/automation/bin/setupRegistryAccess.sh httpperf
    - Find your ingress url and secret
      - armada-perf-client -action=GetCluster -clusterName=YOUR_CLUSTER_NAME | grep ingressHostname
      - armada-perf-client -action=GetCluster -clusterName=YOUR_CLUSTER_NAME | grep ingressSecretName
    - Find the zone and Public VLAN for your workers, for use by the Load Balancer 2 service
      - kubectl describe node <ANY NODE> | grep -E "zone|publicVLAN"
    - Deploy the httpperf application using helm
      - cd /performance/armada-perf/httpperf/imageDeploy directory
      - Install httpperf deployment (replace stage4 in the image name if you wish to use a different stage image)
        - helm install httpperf httpperf --namespace httpperf --set image.name=armada_performance_stage4/httpperf,replicaCount=4,ingress.hosts={YOUR_INGRESS_URL},ingress.tls.hosts={YOUR_INGRESS_URL},ingress.tls.secretName=YOUR_INGRESS_SECRET_NAME,
        service.loadBalancer2.zone="${zone}",service.loadBalancer2.vlanID=\"${vlan}\"

    Note: To delete the httpperf deployment AFTER the tests are complete:
    - helm uninstall httpperf --namespace httpperf

## Running the Tests Manually on a Performance Client Machine

Simple tests can be run using curl. More complex tests can be run using JMeter.
The requests url can either end with '/hello', which returns 'Hello world!', or '/request?size=X', which returns X bytes. e.g <http://X.X.X.X:30079/request?size=1024>

- Finding the ip addresses and ingress urls

  - Find the node ip addresses:
    - kubectl get nodes

  - Find loadbalancer ip
    - kubectl describe service httpperf-lb-service -n httpperf | grep "LoadBalancer Ingress"

  - Find the loadbalancer2 ip
    - kubectl describe service httpperf-lb2-service -n httpperf | grep "LoadBalancer Ingress"

  - Find the ingress url and url app paths:
    - kubectl describe ingress httpperf-ingress -n httpperf | grep -b1 /

- Simple http and https curl testing (size specifies the number of bytes to return. If omitted, 'hello world' is returned)
  - Node Ports: http and https requests examples. Replace X.X.X.X with a 10.X.X.X node port of one of your worker nodes.
    - curl <http://X.X.X.X:30079/request?size=1024>
    - curl -k <https://X.X.X.X:30042/request?size=1024>

  - Load Balancer: http and https requests examples. Replace X.X.X.X with your load balancer ip.
    - curl <http://X.X.X.X:30080/request?size=1024>
    - curl -k <https://X.X.X.X:30043/request?size=1024>

  - Load Balancer 2: http and https requests examples. Replace X.X.X.X with your load balancer 2 ip.
    - curl <http://X.X.X.X:30090/request?size=1024>
    - curl -k <https://X.X.X.X:30053/request?size=1024>

  - Ingress: http and https requests examples. Replace INGRESS_URL with your cluster ingress URL.
    - curl <http://INGRESS_URL:80/request?size=1024>
    - curl -k <https://INGRESS_URL:443/request?size=1024>

- Running tests with JMeter
  - cd /performance/armada-perf/httpperf

  - The httpperf-testplan.jmx file in this directory is used by jmeter to controls how the test is run.
        - This file can be edited by installing and running the JMeter GUI on your laptop.
  - The test is run on the performance client as follows, for example for https ingress:
    - /usr/local/apache-jmeter/bin/jmeter -n -t httpperf-testplan.jmx -JRUNLENSEC=30 -JRAMPUPSECS=0 -f -JTPUTMINS=6000 -JTHREAD=50 -JPORT=80 -JPROTOCOL=http -JHOSTS_FILE=hosts.csv -j httpperf.log -l httpperf.jtl
      - Note the initial log4j stack traces are not a problem

    - The parameters are as follows
      - httpperf-testplan.jmx : the name of jmeter test plan that controls the test. The same test plan file is used for all tests.
      - JRUNLENSEC : the runlength in seconds for the test to run
      - JRAMPUPSECS : the ramp up time of the test - best left at 0
      - JTPUTMINS : the throughput limit (!! requests/MINUTE !!) of each thread.
      - JTHREAD : the number of JMeter threads to run
      - JPORT : The port to send the requests to
        - Node Port http : 30079
        - Node Port https : 30042
        - Load Balancer http : 30080
        - Load Balancer https : 30043
        - Ingress http : 80
        - Ingress https : 443
      - JPROTOCOL: either http or https
      - JHOSTS_FILE the name of the file containing the target address. It must contain one of the following types (NOTE - the JPORT must be set appropriately):
        - the nodeports (one on each new line of the file)
        - the load balancer IP
        - Ingress URL to target.

      - j : The name of the jmeter output log file which contains information about the jmeter threads running and the measured throughput.
      - l : The name of the jmeter output jtl file which contains information about individual http requests and their associated response codes.

## Debugging the Tests

The first point of investigation in the automation is to look at the Jenkins job files to see which tests have failed.

For standalone:

- Log on to the performance client that ran the tests
- cd /performance/armada-perf/httpperf
- Check the the jmeter log files for the failing test(e.g. httpNodePort.log). This file contains information about JMeter threads and throughput.
- Check the the jmeter jtl files for the failing test (e.g. httpNodePort.jtl). This file contains information about each http request and response

For distributed:

- Exec into the jmeter master pod on the load driver cluster to find the log files for investigation.

For small error percentage, check
- `ibm-cloud-provider-ip` can be restarted causing errors
- if `http-perf`/`https-perf` are restarted, that can cause errors

If additional investigation is required, then follow the steps in the "Running the Tests Manually on a Performance Client Machine" section of this readme. This can be used to curl individual protocols and services to test them and facilitate further debugging.
