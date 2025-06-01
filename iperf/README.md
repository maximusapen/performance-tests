# Armada Iperf

This project simplifies the execution of the linux iperf server & client within a Kubernetes cluster.
A single command is exposed that allows configuration of the iperf server & client.

## The Client Machine

It is assumed that one of the Armada Performance client machines will be used to initiate the tests. This machine should already be setup with all the necessary prereqs. See separate documentation for this step.

Once logged onto the client machine, locate the armada-performance git repo. The instructions below assume it has been cloned to the <GITHUB_ROOT>/alchemy-containers directory on the client machine. There should be a directory structure like: GITHUB_ROOT>/alchemy-containers/armada-performance/iperf.

Here, there are three main folders:
* imageCreate: Used to build the docker image,
* imageDeploy: Contains the Helm charts that controls the deployment of the iperf server & client to a cluster
* bin: Contains the script that deploys and runs the iperf server & client on a cluster.

The next step is to build the iperf image and upload it to the registry.

### Building and Uploading the Iperf Image to the Registry

If the desired iperf server image is not already in the registry, build the image using docker and upload it to the registry.  Two options below.

#### Jenkins Job

https://alchemy-containers-jenkins.swg-devops.com/job/Armada-Performance/job/perfClient-setup/job/Publish-Perf-Docker-Images/

#### Manual Steps

These instructions assume you have cloned the armada-performance repo on your client machine.

On the client machine (replace <GITHUB_ROOT> with the location of the alchemy-containers organisation)
```
cd <GITHUB_ROOT>/alchemy-containers/armada-performance
```

Build the docker image locally. (Examine the Dockerfile to see how it builds the image). The image MUST be built from the above directory
```
docker build -t iperf -f iperf/imageCreate/iperf/Dockerfile .
```
Tag the image to point at the appropriate repository and namespace (below uses stage registry and a image version of 1.1 as an example)
```
docker tag iperf stg.icr.io/armada_performance/iperf:latest
```
Push the image to the IBM Cloud registry (example uses the stage registry) after logged into
registry service
```
ibmcloud login -u armada.performance@uk.ibm.com -p <password> -c 4a160c3a25d49f6171b796555191f7da
ibmcloud cr login
docker push stg.icr.io/armada_performance/iperf:latest
```

View the image in the registry
```
ibmcloud cr images
```

## Deploy and run the iperf image in a Kubernetes Cluster

Configure access to your Kubernetes cluster. Note, the iperf client & server can either be run in the same cluster, or seperate ones.
Can use the IBM Cloud CLI, or issue the following commands

```
$GOPATH/bin/armada-perf-client -action=GetClusterConfig -clusterName=<name of your cluster> -admin
unzip kubeConfig-<name of your cluster>.zip

cd ./kubeConfig######### (##'s are a randomly generated set of digits')
export KUBECONFIG=<full path to file>/kube-config-<location>-<<name of your cluster>.yml
```
If you just set KUBECONFIG then the client and server will be installed into the same cluster.
If you want to run the client & server in different clusters then run:
```
export CLIENT_KUBECONFIG=<full path to file>/kube-config-<location>-<<name of your client cluster>.yml
export SERVER_KUBECONFIG=<full path to file>/kube-config-<location>-<<name of your server cluster>.yml
```

If this is a ROKS cluster, you need to login to cluster first.
```
oc login -u apikey -p <STAGE_GLOBAL_ARMPERF_IBMCLOUD_APIKEY>
```

Ensure that the armada performance registry secret has been added to the service account.  Note you will need to get the STAGE_GLOBAL_ARMPERF_REGISTRY_APIKEY from vault/Thycotic to manually run setupRegistryAccess.sh successfully.

```
./automation/bin/setupRegistryAccess.sh <kubernetes namespace>
```

For example,

```
./automation/bin/setupRegistryAccess.sh iperf
```

Note, that the iperf pods have an antiAffinity setting set so that only 1 iperfserver pod & 1 iperfclient will run on each node. You need to ensure you have enough nodes to run all the pods you request.

The iperf3 uses default port 5201.  Armada nodeports allows range 30000 to 32767 only.
By default a node port service iperfserver-np-service-1 is configured to use nodePort 30521. You can deploy multiple services by specifying the --id parameter to the iperfserver.sh script (see below). It will add the value of id to 30520 - to allow many iperfservers to be deployed to a cluster.

### Running Tests
Three bash scripts are provided which will dynamically configure a Kubernetes deployment on a cluster, and then execute the Linux iperf server or client with the supplied options. (Note that a registry token for the desired registry must have been created on the cluster as a Kubernetes secret with the name 'performance-registry-token'. See previous step). You can either run the iperfserver.sh & iperfclient.sh scripts manually to deploy the server & client - or just use the run_iperf.sh script - which can deploy multiple instances of both the server & client on different ports. It will automatically determine the IP addresses and ports on the servers to use for the client.
```
cd <GITHUB_ROOT>/alchemy-containers/armada-performance/iperf/bin

./iperfserver.sh --help

```
The iperfserver.sh script has the following options of its own, together with all options supported by iperf3:
* -g | --registry : The image registry location (e.g. stg.icr.io)
* -n | --namespace : Kubernetes namespace for deployment (e.g. iperfserver)
* -e | --environment : Registry environment namespace (e.g stgiks4 )
* -p | --pods : Number of replicas (pods) of iperf servers
* -i | --id : An identifier that can be used so you can run multiple iperfservers in a single cluster.
* -h | --help : Show help

To use iperf client to connect to the iperfserver using node port service iperfserver-np-service-1, run using port 30521 using private or public IP:

```
./iperfclient.sh --help

```

The iperfclient.sh script has the following options of its own, together with all options supported by iperf:
* -g | --registry : The image registry location (e.g. stg.icr.io)
* -n | --namespace : Kubernetes namespace for deployment (e.g. iperfserver)
* -e | --environment : Registry environment namespace (e.g stgiks4)
* -i | --id : An identifier that can be used so you can run multiple iperfservers in a single cluster.
* -a | --address : The IP address of the iperfserver to connect to.
* -h | --help : Show help

```
./run_iperf.sh --help

```

The run_iperf.sh script has the following options of its own, together with all options supported by iperf:
* -g | --registry : The image registry location (e.g. stg.icr.io)
* -n | --namespace : Kubernetes namespace for deployment (e.g. iperfserver)
* -e | --environment : Registry environment namespace (e.g stgiks4)
* -c, --concurrency : the number of concurrent servers & clients to run
* -h | --help : Show help


##### Examples:

Example 1

Server:
```
./iperfserver.sh -n iperf -p 1
```

ROKS cluster requires public VLAN, add --vlan for the datacenter of cluster.

Public VLANs:

* dal09: 2951366
* dal10: 2912560
* dal12: 2917050
* dal13: 2912590

```
./iperfserver.sh -n iperf -p 1 --vlan <public vlan>
```

The above iperfserver.sh commands will run the iperf server on a single pod in the cluster within the 'iperf' namespace (-n iperf).

Example 1 (client):

Client running outside of the cluster:
```
iperf3 -c $server -p 30521 -P 8 -t 30 -i 2
```
($server is the IP address of one of the nodes in the cluster)

Example 1 (client):

Client running inside the cluster:
```
./iperfclient.sh --id 1 --address $server -t 60
```
(Using --id 1 means it will use port 30521, -t 60 means run the client for 60 seconds)

To see the results, view the logs of iperclient pods:
```
kubectl logs iperfclient-1-job-1-xxxxx -n iperf
```

Example 2

Using run_iperf.sh
```
./run_iperf.sh --registry stg.icr.io --namespace iperf --environment stage1 --concurrency 2
```
will run 2 instances of the iperfserver & 2 instances of iperfclient. Each iperfserver will listen on different ports, and there will be 1 client connecting to each server.


**NOTE:** Any options not included in the list above *MUST* be added at the end.

See the iperf help pages for full information on all options available.

**NOTE:** You will see much higher bandwidth if you hit the NodePort for the node where the iperfserver pod is running, then if you hit a remote Node. Keep this in mind when running tests. Also if runing client & server in the same cluster, you may get the client & server running onthe same node, which will also see significantly higher bandwidth.
