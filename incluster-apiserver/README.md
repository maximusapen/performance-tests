# incluster-apiserver benchmark application

incluster-apiserver uses the Kubernetes client-go to execute a workload against the apiserver for the cluster it is running in.

Each pod makes get pods requests against a specified namespace. It can either be run at a fixed throughput or unlimited.

If the namespace specified exists it will run against that, otherwise it will create a new namespace, with 5 pods

The number of replicas for each "instance" can be scaled up as required. Each "instance" of incluster-apiserver runs in its own Service Account, with associated ClusterRole & ClusterRoleBindings. The number of instances can be scaled up as required.

The total load will be throughput per pod x replicas x instances. At the end of the test the total throughput and response time data for all pods will be merged by the `bin/run_incluster_apiserver.sh script`


## Instructions to deploy/run
The benchmark will usually be run as part of our automation, but the instructions for building manually are included here for convenience.

### Check to see if the incluster-apiserver benchmark image is already in the registry
Log in to IBM Cloud
```
ibmcloud login -a  https://test.cloud.ibm.com (dev/stage)
ibmcloud login -a  https://us-south.containers.cloud.ibm.com (production)
```
Log in to the appropriate registry
```
ibmcloud cr api https://stg.icr.io/api (dev/stage)
ibmcloud cr api https://us.icr.io/api (production)
```
View the current images in the registry:
```
ibmcloud cr images
```
Check that the armada_performance namespace exists (it probably does)
```
ibmcloud cr namespaces
```
Create the armada_performance namespace if it is missing
```
ibmcloud cr namespace-add armada_performance
```
If the required incluster-apisever benchmark image is already available then you can skip the next section on 'Building and Uploading the incluster-apiserver benchmark Image to the Registry'.

### Building and Uploading the incluster-apiserver  benchmark Image to the Registry

If the desired incluster-apiserver  benchmark image is not already in the registry, build the docker image and upload it to the registry.

These instructions assume you have cloned the armada-performance repo on your client machine.

On the client machine (replace <GITHUB_ROOT> with the location of the alchemy-containers organisation)

Build the incluster-apiserver  application locally:
```
cd <GITHUB_ROOT>/alchemy-containers/armada-performance/incluster-apiserver/imageCreate/incluster-apiserver
CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" -a -installsuffix cgo .
cd <GITHUB_ROOT>/alchemy-containers/armada-performance
```

Build the docker image locally. (Examine the Dockerfile to see how it builds the image). The image MUST be built from the above directory
```
docker build -t incluster-apiserver -f incluster-apiserver/imageCreate/incluster-apiserver/Dockerfile .
```
Tag the image to point at the appropriate repository and namespace (below uses stage registry, with the [optional] stage1 namespace and a image version of latest as an example)
```
docker tag incluster-apiserver stg.icr.io/armada_performance[_stage1]/incluster-apiserver:latest (dev/stage)
```

Login and Push the image to the IBM Cloud Registry (example use stage registry)
```
docker login -u iamapikey -p <your_apikey> stg.icr.io
docker push stg.icr.io/armada_performance[_stage1]/incluster-apiserver:latest
```
(Use `STAGE_GLOBAL_ARMPERF_IBMCLOUD_APIKEY` from Vault or your own api key)

View the image in the registry
```
ibmcloud cr images
```
## Run the incluster-apiserver benchmark on a Kubernetes Cluster

The test is run by using the `bin/run_incluster_apiserver.sh` script.
e.g:

`./run_incluster_apiserver.sh -n incluster-apiserver -g stg.icr.io -e stgiks4`

This will use default settings for `throughput`, `replicas` and `instances` but these can be over-ridden on the command line:

`./run_incluster_apiserver.sh -n incluster-apiserver -g stg.icr.io -e stgiks4 -t 5 -i 20 -r 50`

## incluster-deployment test

This is a stripped down modified version of IKS peformance incluster-apiserver test which allows testing create/delete speficied number of deployments and replicas for sysdig testing without actually running the incluster-apiserver test.

### incluster-deployment test instructions

The test is run by using the `run_deployment_test.sh` script in `bin` directory.
e.g:

`./run_deployment_test.sh`

This will use default settings for `replicas` (5) and `instances` (10) but these can be over-ridden on the command line:

`./run_deployment_test.sh -i 20 -r 50`

To get help:

`./run_deployment_test.sh -h`

Note:  DO NOT go too crazy with the instances and replicas as there is a danger of taking down the managed environment.  You should only run this in an environment where you can monitor the cpu and memory of the master.
