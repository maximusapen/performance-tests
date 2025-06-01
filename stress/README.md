# Armada 'stress' utility

These tests simplify the execution of the linux "stress" utility within a Kubernetes cluster.
A single command is exposed that executes the stress utility on a cluster with support for the full set of the linux stress tool's parameters.
It will also assign Kubernetes CPU and/or memory resource requests to the container running within each pod. (Useful for autoscale testing)

## The Client Machine

It is assumed that one of the Armada Performance client machines will be used to initiate the tests. This machine should already be setup with all the necessary prereqs. See separate documentation for this step.

Once logged onto the client machine, locate the armada-performance git repo. The instructions below assume it has been cloned to the <GITHUB_ROOT>/alchemy-containers directory on the client machine. There should be a directory structure like: <GITHUB_ROOT>/alchemy-containers/armada-performance/stress.

Here, there are three main folders:
* imageCreate: Used to build the docker image,
* imageDeploy: Contains the Helm chart that controls the deployment of the stress tool to a cluster
* bin: Contains the script that deploys and runs the stress tool on a cluster.

The next step is to build the stress image and upload it to the registry.

### Building and Uploading the Stress Image to the Registry

If the desired stress tool image is not already in the registry, build the image using docker and upload it to the registry.

These instructions assume you have cloned the armada-performance repo on your client machine.

On the client machine (replace <GITHUB_ROOT> with the location of the alchemy-containers organisation)
```
cd <GITHUB_ROOT>/alchemy-containers/armada-performance
```

Build the docker image locally. (Examine the Dockerfile to see how it builds the image). The image MUST be built from the above directory
```
docker build -t stress -f stress/imageCreate/stress/Dockerfile .
```
Tag the image to point at the appropriate repository and namespace (below uses stage registry and a image version of 1.1 as an example)
```
docker tag stress stg.icr.io/armada_performance/stress:latest
```
Push the image to the IBM Cloud registry (example uses the stage registry)
```
docker push stg.icr.io/armada_performance/stress:latest
```

View the image in the registry
```
ibmcloud cr images
```
## Deploy and run the Stress tool in a Kubernetes Cluster

Configure access to your Kubernetes cluster.
Can use the IBM Cloud CLI, or issue the following commands

```
$GOPATH/bin/armada-perf-client -action=GetClusterConfig -clusterName=<name of your cluster> -admin
unzip kubeConfig-<name of your cluster>.zip

cd ./kubeConfig######### (##'s are a randomly generated set of digits')
export KUBECONFIG=<full path to file>/kube-config-<location>-<<name of your cluster>.yml
```
Ensure that the armada performance registry secret has been added to the default service account for the namespace

```
$GOPATH/bin/automation/bin/setupRegistryAccess <namespace>
```

### Running Tests
A single bash script is provided which will dynamically configure a Kubernetes deployment on a cluster, and then execute the Linux stress utility wii the supplied options. (Note that a registry token for the desired registry must have been created on the cluster as a Kubernetes secret with the name 'performance-registry-token'. See previous step)
```
cd <GITHUB_ROOT>/alchemy-containers/armada-performance/stress/bin

./stress-driver.sh --help

```
The stress-driver.sh script has 8 options of its own, together with all options supported by the Linux stress utility:
* -g | --registry : The image registry location (e.g. stg.icr.io)
* -n | --namespace : Kubernetes namespace for deployment (e.g. stress)
* -e | --environment : Registry environment namespace (e.g stgiks4)
* -p | --pods : Number of replicas (pods) within which the tests are to be run
* -c | --cpu : Number of workers (CPU cores) to request and load per pod
* -m | --vm : Number of workers (CPU cores) to perform memory load per pod
* --vm-bytes : Amount of memory to request and malloc per vm worker on each pod (default is 256MB)
* -h | --help : Show help

##### Examples:

Example 1
```
./stress-driver.sh -n stress -p 1 -c 1
```
will run the stress utility on a single pod in the cluster within the 'stress' namespace (-n stress), the pod requesting and utilising 1 CPU core for CPU stress testing.

Example 2
```
./stress-driver.sh -n stress -p 2 -c 1 -m 1 --vm-bytes 500M
```
will run the stress utility on 2 pods in the cluster within the 'stress' namespace (-n stress), each pod requesting and utilising 1 CPU core for CPU stress testing and an additional 1 CPU core for memory stress testing (malloc()/free() 500MB of memory)

**NOTE:** Any Linux stress utility options not included in the list above *MUST* be added at the end.

See the Linux stress utility help pages for full information on all options available.
