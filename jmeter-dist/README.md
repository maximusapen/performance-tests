# Armada JMeter distributed testing

JMeter distributed testing supports load testing of an external HTTP(S) server using load drivers running within pods within a Kubernetes cluster. It will use JMeter's distributed testing or use independent JMeter instances depending if --mode is set to "slave" or "standalone" respectively.

For an overview of JMeter distrubted testing see https://jmeter.apache.org/usermanual/jmeter_distributed_testing_step_by_step.pdf

## Images

There are 4 main images:
* base  
The base image which contains a JMeter installation used by the master/slave/standalone images.  

* master  
The JMeter master controller image. This receives requests from a user (via a TCP socket) and then initiates a test on one or more slave pods.  

* slave  
The JMeter Slave image. This receives commands from the master controller pod and sends requests to the target system(s)

* standalone  
The JMeter Standalone image. This receives commands from the master controller pod and sends requests to the target system(s)

### Building

To build the images, run the following command:  

```./bin/build.sh [--environment]```  

options:  
-e | --environment
> Specifies the registry namespace to be used, e.g. 'stage2' -> 'armada_performance_stage2'.  
Default value is 'armada_performance'. 

This will build the docker images and push them to the _stage_ container registry.  

## Installation

Prior to installation, ensure that the KUBECONFIG environment variable is configured for use with your load driver cluster.
 
To install the distributed JMeter load driver on an IKS cluster, run:  
```./bin/install.sh [options] testname```  

options:  
-h, --help
> show brief help

-t, --threads <N>
> default number of JMeter threads to be run on each slave pod

-r, --rate <N>
> target throughput of each thread in requests/second. If not specified, the value from the test _.jmx_ file wil be used.

-d, --duration <N>
> default test duration in seconds. If not specified, the value from the test _.jmx_ file wil be used.

-n, --namespace <k8s_namespace>
> Kubernetes namespace for deployment

-p, --password <keystore_password>
> Password for accessing the Java keystore containing the target server authentication credentials

-s, --slaves <N>
> number of JMeter slave replicas  (pods) which are to be used as JMeter load generators

-e, --environment <environment>
> registry environment namespace (e.g stage1, stage2, etc., defaults to stage1)

-m, --mode [slave|standalone]
> Defines the pod deployment mode (slave is the default).

_**testname**_ is the name of the directory that hosts the test configuration files.  It should be located under ```./tests``` and contain the following files:  

* cert.jks  
java keystore containing certificates for authentication with target server.  
* clusters.csv  
a csv file containing the remote target server connection information  
```id,scheme,address,port```  
* requests.csv  
a csv file containing the request(s) to be made against the target server.  
```action,path,[body]```  
* test.jmx  
It is recommended that the JMeter test configuration file in the "example" testcase is used. Any significant modifications may result in a failure to run the test correctly.

(N.B. the "example" test case and associated configuration files can be found under ```./tests/example```)  

See https://github.ibm.com/alchemy-containers/armada-performance/blob/master/k8s-apiserver/README.md for further details of the file formats and specifically, how to generate files for targetting an IKS cluster's api server.  


Note that ```install.sh``` is a simplified wrapper around
```./bin/jmeter-driver.sh```  using default values for a number of parameters.

jmeter-driver - runs Kubernetes based distributed JMeter load testing
 
jmeter-driver [options] args
 
options:  
-h, --help
> show brief help

-c, --config <folder>
> folder containing runtime configuration files. Defaults to ```./config```.

-t, --threads <N>
> number of JMeter threads to be run on each slave pod

-r, --rate <N>
> target throughput of each thread in requests/second

-d, --duration <N>
> test duration in seconds

-e, --environment <environment>
> registry environment namespace" (e.g stage1, stage2, etc.)

-g, --registry <registry_url> 
> image registry location

-n, --namespace <k8s_namespace>
> Kubernetes namespace for deployment

-p, --password <keystore_password>
> Password for accessing the Java keystore containing the target server authentication credentials

-s, --slaves <N>
> number of JMeter slave replicas (pods) which are to be used as JMeter load generators.  

> N.B. If not specified, each slave pod will request 1 cpu core, and the maximum number of replicas that can be spread evenly across all cluster nodes will be created, typically (number of node cpu cores - 1) per node.

## Running tests  

To run tests:  
```./bin/run.sh [--threads] [--rate] [--duration]```  

* -n | --namespace  
Kubernetes namespace within which the application was deployed.

* -t | --threads  
Number of JMeter worker threads to be run on each slave pod. Default value specified at [installation](#installation) time.

* -r | --rate  
Target throughput of each thread in requests/second. Default value specified at [installation](#installation) time.

* -d | --duration  
Test run length. Default value specified at [installation](#installation) time.

This script will send the test START command to the master-controller pod, sleep for the duration of the test and then request the results from the master-controller pod, which are then echo'ed to stdout.

Alternatively, tests can be triggered manually using:

```echo -en "START" | nc <cluster_worker_node_ip> 30444```  
 
and then the test results obtained with:  

```echo -en "" | nc <cluster_worker_node_ip> 30444```
