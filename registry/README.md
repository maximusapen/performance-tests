# Armada Registry Performance Tests

These tests use a GO client to dynamically push and pull a 50Mb image in various ways using an image deployed into a Kubernetes cluster in the targeted region (us-south, au-syd, eu-gb, eu-de,ap-north) :
* To and from the local regional registry (in prod)
* To and from the International registry
* To and from each of the regional registries from a cluster in us-south
The timings are taken and published to the stage metrics service. These test run on a Lite cluster (patrol) in Prod in each region.

These tests are run automatically every day at 7am and 2pm in all regions using a jenkins job:

https://alchemy-testing-jenkins.swg-devops.com/view/Armada-performance/job/Armada-Performance/job/Automation/job/Run_registry_tests/

Using Jenkins is the easiest way to run the tests but if you want to manually run them follow these instructions.

## The Client Machine

It is assumed that one of the Armada Performance client machines will be used to initiate the tests (Jenkins is set up to use stage-dal09-perf1-client-03).  This machine should already be setup with all the necessary prereqs but if not then use Jenkins jobs under here to setup: https://alchemy-containers-jenkins.swg-devops.com/job/Armada-Performance/job/perfClient-setup/

Once logged onto the client machine there should be a directory structure like: /performance/armada-perf/armada-performance/registry/

Here, there are three main folders:
* imageCreate: Used to build the docker image and also contains the individual tests.
* imageDeploy: Contains the Kubernetes Job configuration file
* bin: Contains the script to run the tests.

The next step is to build the registry image and upload it to the registry if needed.

### Building and Uploading the Image to the Registry

If the desired image is not already in the registry, build the docker image and upload it to the registry using this Jenkins job:
https://alchemy-containers-jenkins.swg-devops.com/job/Armada-Performance/job/perfClient-setup/job/Publish-Perf-Docker-Images/

## Setup your Kubernetes Cluster

(The Jenkins tests use a cluster called regtest-xxx in each region which is already setup and can be used rather than creating your own.)

Log in to ibmcloud in the region you want to test and select the registry-dev account.
```
ibmcloud login -a https://cloud.ibm.com -r us-south (to test in us-south)
```
Log in to the appropriate registry
```
ibmcloud cr login
```
These instructions assume kubectl is installed on the client machine.

Configure Kubernetes Client (kubectl) to access your Kubernetes cluster.
Can use the IBMCloud CLI (ibmcloud ks cluster config <name of your cluster>), or issue the following commands

```
$GOPATH/bin/armada-perf-client -action=GetClusterConfig -clusterName=<name of your cluster>
unzip kubeConfig-<name of your cluster>.zip

cd ./kubeConfig######### (##'s are a randomly generated set of digits')
export KUBECONFIG=<full path to file>/kube-config-<location>-<<name of your cluster>.yml

e.g. export KUBECONFIG=kube-config-par01-regtest.yml
```

### Run the Tests
A bash script is provided which will dynamically configure a Kubernetes job, using a Kubernetes config map and environment variables.
```
cd <GITHUB_ROOT>/alchemy-containers/armada-performance/registry/bin

./perf-registry.sh --help
```
The perf-registry.sh script accepts these options:
* -h, --help : show brief help
* -k, --registrykey key : IBM registry access api-key
* -e, --environment environment : registry environment namespace (e.g dev9, stage1, etc.)
* -g, --registry registry_url :	image registry location
* -r, --regional registry_url :	regional registry location you want to test
* -n, --namespace k8s_namespace : kubernetes namespace for deployment
* -m, --metrics : send results to metrics service
* -v, --verbose : log test results to stdout
* -i, --international : run against international registry
* -l, --allRegions : run against all regional registries from one cluster
* -c, --clusterRegion : cluster region to use if --allRegions specified

For example, this command will use helm to deploy the registry image "stg.icr.io/armada_performance_stage1/registry" which then runs the push/pull test against the us-south registry at "us.icr.io". Find the apikeys from vault.
```
./perf-registry.sh -v -e "stage1" -n "registry" -g "stg.icr.io" -k <PROD_GLOBAL_ARMPERF_IBMCLOUD_APIKEY> -r "us.icr.io"
```
To monitor the tests, you can view the docker logs (assuming -v option was specified)
```
kubectl get pods -n registry
```
```
kubectl logs <pod name> -n registry
```
To obtain test results, either:

* View the results sent to the grafana dashboard under the page '_registry'

* Get results from docker logs (assumes -v option was selected)
```
kubectl logs <pod name> -n registry > test_results.out
```
