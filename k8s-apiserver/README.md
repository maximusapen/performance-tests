# Kubernetes API Server Load testing

The resources here contain a JMeter test plan and tools to dynamically generate the associated configuration files for load testing of Kubernetes API server(s) across one of more previously created clusters. The clusters do not necessarily require worker nodes, unless the requests would result in activity on a worker node (e.g. create deployment)

## Resources

Main files are:

* K8SAPIServer-direct.jmx  
  A JMeter test plan.  
  The test plan uses an HTTP Request sampler, with an associated HTTP Header manager to issue requests with a target throughout to the api servers on a set of clusters. A keystore configuration element is used to reference the required authentication credentials.  
  See [Running] section for more details.

* api-loadtest-config.sh  
  Before the tests can be run, cluster configuration details must be generated using this bash script.  
  (All output will be generated in `<armada-performance-repo-root>/k8s-apiserver/jmeter-config`)  

  To run:  

  `./api-loadtest-config.sh <cluster-name-prefix> <keystore-password> [<number-of-clusters>]`  

  * \<cluster-name-prefix> : identifies the cluster(s) which are part of the load testing set.
  * \<keystore-password> : a password that is used to protect the generated java keystore and intermediary certificates. Choose a value.
  * \<number-of-clusters> : an optional parameter that overrides the default number of clusters (all clusters matching the cluster name prefix) which are to be included in the load testing

  This utility will generate the configuration needed by the JMeter test plan. It performs two main roles:  
  1. Generates the `cluster.csv` file which contains the api server connection information for each cluster matching the supplied prefix. Note that all clusters matching the prefix should be number sequentially, starting from 1, that is, cluster1, cluster2, cluster3, etc.  
  Output is \<cluster-name>,\<protocol>,\<hostname>,\<port>. For example:  

      rgszero1,https,c2.stage-dal09.containers.test.cloud.ibm.com,26562  
      rgszero2,https,c2.stage-dal09.containers.test.cloud.ibm.com,25365  
      rgszero3,https,c2.stage-dal09.containers.test.cloud.ibm.com,30662  
      rgszero4,https,c2.stage-dal09.containers.test.cloud.ibm.com,26095  
      rgszero5,https,c2.stage-dal09.containers.test.cloud.ibm.com,28693  

      (N.B. the cluster name in the above output will be (and needs to be) lower case due to limitations in Java keystore certificate aliases)  

  2. Converts the client certificate and key used to authenticate with the api server to a Java keystore format required by JMeter  
  The Java keystore will be generated in `certs.jks`

* request.csv  
  User generated list of requests to send to Kubernetes api server. It is a comma separated list of requests in the following format  
  \<METHOD>,\<REQUEST>,\[\<BODY_FILE>]  
  * METHOD is an http method, e.g. one of GET, POST, PUT, PATCH, DELETE etc.  
  * REQUEST is the http request and parameters, e.g. /api/v1/namespaces/{namespace}/pods/{name}  
  * BODY_FILE is the optional filename (without extension) that contains the request body in json format, e.g. create_namespace  

  For example:  
  * `GET,/api/v1/pods,`  
  * `POST,/api/v1/namespaces,create_namespace`  
  * `DELETE,/api/v1/namespaces/some_namespace,delete_namespace`  

  A useful tip to convert a kubectl command to an equivalent REST API request, is to use the -v8 option on the kubectl command  
  e.g. `kubectl -v8 get pods`  

* crd_requests.csv
* create_namespace.json
* delete_namespace.json  
  An example of requests and associated request body contents that create and delete resources. However.....  

  **IMPORTANT:** Note that currently it is recommended that only read requests are generated. Whilst use of create, delete and update requests is supported, due to the asynchronous manner in which these requests are processed, there is no guarantee that the creation request will have been completed, before an attempt to update or delete is sent. Thus error responses may be returned from the api server.

## Running

`/usr/local/apache-jmeter/bin/jmeter -n -t K8SAPIServer-direct.jmx -JRUNLENSEC=300 -JRAMPUPSECS=0 -JTPUTMINS=1200 -JTHREAD=50 -o report -Djavax.net.ssl.keyStore=./cert.jks -Djavax.net.ssl.keyStorePassword=abcdef -f -e -l samples.out`  

* -JTHREAD : Number of threads (users) with which to issue requests
* -JRUNLENSEC : Test duration (in seconds)
* -JRAMPUPSECS : How long to ramp up to the maximum number of threads (in seconds)
* -JTPUTMINS : Target throughput per thread in requests/min  (_Test plan uses a 'Constant Throughput Timer'_)

See [JMeter documentation](http://jmeter.apache.org/usermanual/get-started.html) for further information
