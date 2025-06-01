# Armada Persistent Storage Performance Tests

These tests run a Kubernetes job that uses a simple GO client within a debian based image, to dynamically drive a couple of Linux utilities to measure the performance of persistent storage within a Kubernetes cluster. For classic clusters, this can be either file or block storage. For VPC clusters, only block storage is supported.


The utilities are:

* fio : Measures read/write performance
* ioping : Measures io latency


## The Client Machine

It is assumed that one of the Armada Performance client machines will be used to initiate the tests. This machine should already be setup with all the necessary prereqs. See separate documentation for this step.

Once logged onto the client machine, locate the armada-performance git repo. The instructions below assume it has been cloned to the <GITHUB_ROOT>/alchemy-containers directory on the client machine. There should be a directory structure like: <GITHUB_ROOT>/alchemy-containers/persistent-storage.

Here, there are three main folders:
* imageCreate: Used to build the docker image
* imageDeploy: Contains the Helm chart to deploy the Kubernetes Job
* bin: Contains the script to run the tests.

The next step is to build the persistent-storage benchmark image and upload it to the registry.

### Check to see if the persistent-storage benchmark image is already in the registry
Log in to the IBM Cloud stage environment
```
ibmcloud login -a test.cloud.ibm.com

```
Target the appropriate registry
```
ibmcloud cr api https://stg.icr.io/api
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
If the required persistent-storage benchmark image is already available then you can skip the next section on 'Building and Uploading the persistent-storage benchmark image to the Registry'.

### Building and Uploading the persistent-storage benchmark image to the Registry

If the desired persistent-storage benchmark image is not already in the registry, build the image and upload it to the registry.

These instructions assume you have cloned the armada-performance repo on your client machine.

On the client machine (replace <GITHUB_ROOT> with the location of the alchemy-containers organisation)
```
cd <GITHUB_ROOT>/alchemy-containers/armada-performance
```

Build the docker image locally. (Examine the Dockerfile to see how it builds the image). The image MUST be built from the above directory
```
docker build -t persistent-storage -f persistent-storage/imageCreate/persistent-storage/Dockerfile .
```
Tag the image to point at the appropriate repository and namespace  
```
docker tag persistent-storage stg.icr.io/armada_performance/persistent-storage:latest
```
Push the image to the IBM Cloud stage registry
```
docker push stg.icr.io/armada_performance/persistent-storage:latest
```

View the image in the registry
```
ibmcloud cr images
```
## Deploy and run the persistent-storage benchmark on a Kubernetes Cluster

These instructions assume kubectl is installed on the client machine.

Configure Kubernetes Client (kubectl) to access your Kubernetes cluster.
Can use the IBM Cloud CLI, or issue the following commands

```
$GOPATH/bin/armada-perf-client2 cluster config --cluster <name of your cluster> --admin
unzip kubeConfig-<name of your cluster>.zip

cd ./kubeConfig######### (##'s are a randomly generated set of digits')
export KUBECONFIG=<full path to file>/kube-config-<location>-<<name of your cluster>.yml

e.g. export KUBECONFIG=kube-config-dal09-PerfStorageDriverCruiser.yml
```

Ensure that the armada performance registry secret has been added to the default service account for the namespace

```
$GOPATH/bin/automation/bin/setupRegistryAccess <namespace>
```

### Block Storage Configuration
If testing of block storage is required then it is necessary to install the IBM Cloud Block Storage plugin.

The steps required are dependent on the type of cluster
#### Classic:
1. Create a secret containg the global registry read access token in the kube-system namespace. A script (https://github.ibm.com/alchemy-containers/armada-performance/blob/master/automation/bin/setupGlobalRegistryAccess.sh) is provided to assist, but it is necessary to retrieve the token required by the script from https://github.ibm.com/alchemy-1337/environment-stage-dal09/blob/master/armada-performance/armada_performance_id

2. Fetch the IBM Cloud block storage plugin Helm chart
```
helm repo update
helm fetch ibm/ibmcloud-block-storage-plugin --untar --untardir=/tmp
```

3. Add the secret created in step 1 to the service account
```
sed -i '/imagePullSecrets:/a \ \ - name: performance-global-registry-read' /tmp/ibmcloud-block-storage-plugin/templates/ibmcloud-block-storage-plugin-SA.yaml
```

4. Install the plugin
```
helm install ibmcloud-block-storage-plugin /tmp/ibmcloud-block-storage-plugin
```

### Cloud Object Storage Configuration

The Cloud Object Storage resource and bucket is configured in the Armada Performance Production account. Stage cannot currently be used as a paid account is required. A single bucket and associated HMAC credentials are used in the test. The HMAC credentials are stored in vault.  
See https://cloud.ibm.com/objectstorage/crn%3Av1%3Abluemix%3Apublic%3Acloud-object-storage%3Aglobal%3Aa%2F641f32b9227848fdd5d2ab94f1ab4343%3A7ad6b641-4ba6-436f-a9c7-9e61ff6c11fa%3A%3A?paneId=manage

1. Install the plugin  
See https://test.cloud.ibm.com/docs/containers?topic=containers-object_storage#install_cos for details

#### VPC:
1. Install cluster addon (if not already enabled). Note that it can take several minutes for this action to complete.
```
$GOPATH/bin/armada-perf-client2 cluster addon enable vpc-block-csi-driver --cluster <name of cluster>
```

### Running Tests
A bash script is provided which will dynamically configure a Kubernetes job, using a Kubernetes config map and environment variables.
```
cd <GITHUB_ROOT>/alchemy-containers/armada-performance/persistent-storage/bin

./perf-persistent-storage.sh --help

```
The perf-persistent-storage.sh script accepts the following options:
* -h, --help                  	show brief help
* -g, --registry registry_url	image registry location
* -n, --namespace k8s_namespace	kubernetes namespace for deployment
* -m, --metrics			request results are sent to IBM Cloud monitoring service
* -v, --verbose			log test results to stdout
* -j, --jobfile fiojobfile	filepath of fio job file
* -s, --size size			persistent storage size (e.g. 20Gi)
* -c, --class storage_class	persistent storage class name (e.g. ibmc-block-bronze or ibmc-vpc-block-5iops-tier)

For example,
```
./perf-persistent-storage.sh  -v  -n <namespace> -m -c "ibmc-file-silver" -s "100Gi" -j "/tmp/fiojobfile"
```
will run the benchmark against a 100GB volume with the Silver storage class against a classic cluster. Results metrics will be written to a file within the pod, which can be read using automation.


To obtain test results:
* Get results from job logs
```
kubectl logs job/<job name> -n <namespace> >> test_results.out
```
