# Overview of the Armada Performance Tests

This document gives a high level overview of the metrics displayed on the performance dashboard : [Performance Test Suite Results](https://alchemy-prod.hursley.ibm.com/stage/performance/grafana)
___
___

## Cluster Creation Tests
``` Objective```
 * These tests record the time to create and delete an Armada cluster including masters and workers.

```Details```
* The test records the time spent in each cluster creation creation state (requested, provision_pending, provisioning, waiting for master, deploying and deployed) through to full cluster completion. 

```Charts```

* The chart shows the time taken to create a new cluster with 5 workers.

* Lower numbers show better performance.
___
___

## Kubernetes End to End Tests
``` Objective```
 * These tests measure kubernetes performance and are built from the Kubernetes community E2E tests. See https://github.com/kubernetes/community/blob/master/contributors/devel/e2e-tests.md

```Details```
* Kubernetes load performance tests: This test creates 30 pods per node using replication controllers. These tests do not have secrets, configmaps or daemons. The workload consists of several replication controllers with a mixture of 5 or 30 pods each. 
* The test records the time to create, delete and scale pods and services. 
 * The test also includes api response times for: {put, post, list, get and delete} operating on {pods, nodes, services, deployments and replication controllers}

```Charts```

The charts show the API response time to

* put, post, list, get and delete `Pods`
* get and list `Nodes`
* post, list, get and delete `Services`
* get `Deployments`
* put, post, list, get and delete `Replication Controllers`

A chart also shows the time taken to complete each phase of the test including the initial creation of 30 pods per node, the scaling of those pods up and down (twice) and the deletion of those pods. 
___
___
## Pod Scaling Tests
``` Objective```
 * These tests measure the time to scale up/down the number of replicas.

```Details```
* The tests starts with a single pod and scales up using `kubectl scale --replicas=xxx`. Pods are considered to be started when they are in 'Running' state. The pods are then scaled back to one pod.

```Charts```

* The charts show the time taken to scale from 1 to 250 replicas and from 1 to 500 replicas and the time to scale back to 1 replica.

* Lower numbers show better performance.
___
___

## HTTP Tests
``` Objective```
 * These tests evaluate the http and https performance of containers using Nodeports, LoadBalancers, LoadBalancer2 and Ingress.

```Details```
* The client is jmeter running running in distributed mode, with 11 driver nodes by default (1 master & 10 slaves) which sends http and https requests with different numbers of threads. All request sizes are a few bytes but the response payload can be a few bytes or 1KB (50% of the time for each case). 

* The server is a GO http server running in a container. One server is deployed to each Cruiser node. The incoming requests are distributed across the containers using NodePorts, a LoadBalancer or Ingress. 

```Charts```

* The charts show the throughput, response time and % error responses with different numbers of jmeter threads. There is a set of charts for Nodeports, LoadBalancers and Ingress using http and https. 

* Higher numbers show better performance for the throughput (Requests/s).
* Lower numbers show better performance for the response times.
* Zero is the expected result for '%Request error' charts.
___
___

## HTTP Scale: In-Cluster Tests
``` Objective```
 * These tests were used by the Kubernetes community to demonstrate high scalability. The tests create multiple vegeta drivers and send http requests to multiple nginx drivers - all within a single cluster. The aim is to generate a large amount of internal traffic to verify it can handle the load. It is based on this test: https://github.com/kubernetes/contrib/tree/master/scale-demo. 


```Details```
* The vegeta http drivers are distributed in pods across the cluster. The http requests are sent to nginx webservers also distributed in pods across the cluster. An aggregator pod collects the throughput data from all of the vegeta drivers to give a total request rate for the entire cluster.  

```Charts```

* The charts show the throughput and response time for the throughput aggregated across all vegeta drivers. 

* Higher numbers show better performance for the throughput (Requests/s).
* Lower numbers show better performance for the response times.
___
___
## Network Tests
``` Objective```
 * These tests measure Kubernetes network performance and are built from the Kubernetes network performance benchmark. See https://github.com/kubernetes/perf-tests/tree/master/network/benchmarks/netperf

```Details```
* The tests use  containers as workers, each with iperf and other tools built in. Workers 1 and 2 are placed on one node and Worker 3 is placed on a separate node. This allows the following network paths to be tested:

  - Same VM : Worker 1 sending to Worker 2 using PodIP or Cluster IP  
  - Remote VM : Worker 3 sending to Worker 2 using PodIP or cluster IP  
  - Same VM Pod HairPin : Worker sends to itself using Cluster IP  

```Charts```

* The charts show the throughput in Gbits/s

* Higher numbers show better performance

* The charts show:
  - Throughput using TCP  
  - Throughput using UDP
  - Throughput using the netperf benchmark running TCP tests   

___
___
## Sysbench Tests
``` Objective```
 * Ensure that the underlying Cruiser workers associated with the test cluster are performing as expected. Any degradation in the worker performance will impact all other performance tests, so we need to validate their performance. 

```Details```
*   The tests run a series of cpu, memory and file io tests on each worker to measure performance.

```Charts```

* The charts show the time per request for a series of cpu, memory and disk operations with different nubers of test threads. 

* Lower numbers show better performance for the time per request metrics.
___
___

## Container Storage Tests
``` Objective```
 * These tests measure the container storage performance, including the container storage drivers e.g overlay2. Originally based on https://github.com/chriskuehl/docker-storage-benchmark.

```Details```

* The test runs in a single container and runs multiple threads (instances), each one reading or writing to the local docker storage. All the threads perform the same action in a given test. Each test either reads or writes to big files, file trees or small files.

```Charts```

* The charts show the time taken for 10 or 50 threads (instances) to perform a given number of reads and writes.

* Lower numbers show better performance.

 * There are 6 tests:
   - Appending to big files
   - Appending to file trees
   - Appending to small files
   - Reading from big files
   - Reading from file trees
   - Reading from small files.  

___
___

## Persistent Storage Tests
``` Objective```
 * These tests run a Kubernetes job that uses a simple GO client within a debian based image, to dynamically drive a couple of Linux utilities to measure the performance of persistent storage within a Kubernetes cluster.

```Details```
* The Kubernetes job runs the following tests with block and file persistent storage
  * fio : Measures read/write performance
  * ioping : Measures io latency

```Charts```
* Higher numbers show better performance for the IOPS and Bandwidth charts
* Lower numbers are better for the latency charts.
___
___

## AcmeAir Tests
``` Objective```
 * This benchmark emulates a fictional airline called 'Acme Air'. The benchmark represents multiple users concurrently interacting with the airline. The benchmark code is here: https://github.com/blueperf. 

```Details```
* The customer requests are generated using a jmeter http client. The requests are used to to book, cancel or review flights. Each request interacts with one or more of the benchmark's microservices:
  - authorization
  - customer service
  - booking
  - flight
  - main service   
 * The microservices run in pods in the Cruiser and interact with associated databases.

```Charts```
* Higher numbers show better performance for the throughput charts 
* Lower numbers are better for the latency charts.
___
___

## AcmeAir Istio Tests
``` Objective```
 * This is similar to the AcmeAir test, but Istio is installed, and requests are routed through Istio.

```Details```
* The test runs in the same way as AcmeAir, but requests are all routed through Istio.

```Charts```
* Higher numbers show better performance for the throughput charts 
* Lower numbers are better for the latency charts.
___
___

## Online Banking Tests
``` Objective```
 * This benchmark emulates a fictional online bank. For more information see https://github.ibm.com/ibmperf/olb-java/tree/helm .

```Details```
* The application is a simple Java application that simulates front end application to communicate with backend on-premise database server. The helm chart also contains a sub-chart called stub which simulates backend transaction with set sleep time before it returns the data. This test also uses the Kubernetes pod autoscaler to automatically scale up the number of pods during the test.
 * The microservices run in pods in the Cruiser and interact with associated databases.

```Charts```
* Higher numbers show better performance for the throughput charts 
* Lower numbers are better for the latency charts.
___
___

## Zero Worker Tests
``` Objective```
 * Measure the time to deploy masters, and the performance of the masters once created.
 
```Details```
* This test creates 10 clusters that each have zero worker nodes. This enables us to measure the time to deploy masters. Once the masters are deployed it runs the APIServer load tests against the 10 masters. In this test the APIServer load tests are run from a standalone jmeter driver running on a 16 core perf client host. This will measure the throughput and response times from the masters.

```Charts```
* Lower numbers are better for the cluster creation time charts
* Higher numbers show better performance for the throughput charts 
* Lower numbers are better for the latency charts.
___
___

## APIServer Load Tests
``` Objective```
 * Measure the performance of the IKS master apiservers.
 
```Details```
* This test creates 10 clusters that each have zero worker nodes. Once the masters are deployed it runs the APIServer load tests against the 10 masters. In this test the APIServer load tests are run using a distributed load driver, using 11 driver nodes by default (1 master & 10 slaves). This will measure the throughput and response times from the masters.
The load driver makes GET pods & GET namespaces requests against the master apiservers.

```Charts```
* Lower numbers are better for the cluster creation time charts
* Higher numbers show better performance for the throughput charts 
* Lower numbers are better for the latency charts.
___
___


## Hollow Node Tests
``` Objective```
 * These tests create a Cruiser with 200 'hollow' worker nodes. Hollow nodes are fake nodes that are treated as real nodes by kubernetes. For example, the hollow-kublet emulates a real kublet but doesn't actually start any containers. The master sees this as a real Kubelet.
 * This enables tests to be run (emulated) at scale without the cost of creating real nodes. 
 
```Details```
* After creating the hollow nodes, the Kubernetes end load tests (see previous description of tests), which create, scale and interact with 30 pod per node, are run to measure performance.
 

```Charts```

The charts show the API response time to

* put, post, list, get and delete `Pods`
* get and list `Nodes`
* post, list, get and delete `Services`
* get `Deployments`
* put, post, list, get and delete `Replication Controllers`

A chart also shows the time taken to complete each phase of the test including the initial creation of 30 pods per node, the scaling of those pods up and down (twice) and the deletion of those pods. 

___
___

## Registry Tests ##
``` Objective```
 * These tests measure the time taken to upload and download images from the Regional and Global registries.

```Details```
* The tests run in containers on production Cruisers in a number of regions around the world e.g. US-South, UK, Germany, Sydney, Tokyo. In each case the test pushes and pulls a 50MB image (single layer) to the local regional registry and the global registry. It then pulls the Hyperkube 1.8.6 image (512MB but with multiple image layers) from the regional registry. The local registry tests in the ap-north spoke (jp-tok) target the au-syd registry as there is no jp-tok registry. Tests are run twice a day.

```Charts```

* The charts show the time taken to push and pull images plotted against the date of the test.

* Lower numbers show better performance.

___
___
