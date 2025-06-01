# Cruiser Capacity tool
Tools to examine cruiser capacity on resource limitations

The deployTemplate.yaml provides deployment of a pod with 6 containers and
3 services which has a total of 5 endpoints.  There are 4 exec liveness probes
and 2 http liveness probes for the pod.

You can remove containers, services and probes from deployTemplate.yaml as you
wish for your testing.  If deleting services, you may want to remove the
HTTPPERF_*_HTTP* assignment that you don't need so more ports are available for
your testing.  No harm to leave them if not bothered.


## Test Preparation

### SSH access to workers

Set up privileged ssh access to all cruiser workers.  Use tools/sshCruiserNodes.

### Create namespace

Create the namespace and add registery token to the namespace before testing a
new cruiser cluster.

Run commnand in cruiserPod directory to create the first httpperf1 namespace with
the registry token:

    ./initNamespace.sh

Then clone the namespace httpperf1 to additional httpperf*.
To have a total of 100 httpperf namespaces:

    ./cloneNamespace.sh 2 100

It is a lot quicker to cloneNamespace then to create each one individually
when registry token is involved.


### Gather PLEG data from nodes

Pleg and meminfo are collected from each worker.

Scripts:
- pleg.sh for Dockerd
- pleg-cri.sh for Containerd
- meminfo.sh for both

2 installation scripts are provided to copy the pleg script to
/root/pleg/pleg.sh and meminfo.sh to /root/mem directory on all workers.

Run command as
- For Dockerd:

    ./install-pleg.sh < cluster-name >

- For Containerd:

    ./install-pleg-cri.sh < cluster-name >

You do need to ssh into each worker to start the pleg.sh and meminfo.sh:
- In /root/pleg:

    nohup ./pleg.sh &

- In /root/mem

    nohup ./meminfo.sh &

To retrieve the pleg and meminfo data from all workers, Run command:

    getplegfiles.sh < cluster-name >

### Monitor PLEG issues external to nodes

A script checknodes.sh is provided to check for NotReady nodes and not Running
pods for the cruiser cluster every 60 sec.

Run the script as:

    nohup <path to>/checknodes.sh &

Check notready.log for any PLEG issues.

Remember to kill checknodes.sh process after test.

### Create test pods

To create pods on the cruiser cluster, set up your KUBECONFIG to your cruiser
cluster, then run command in cruiserPod directory:

    ./createtestpods.sh <pod start number> <pod end number> <service port start>

Example:
To create 1 test pod in 1 namespace:

    ./createtestpods.sh 1 1 30000

To create 100 pods in 100 namespaces:

    ./createtestpods.sh 1 100 30000

The service port needs to be in range of 30000 and 32767.

If you run the tests with pods created in batches and start with 30000 for the
<service port start>, console output will display the next port number to use
with "next port <port numner to use next>"

By adjusting periodSeconds for the exec livenessProbe in deployTemplate.yaml,
now set at 35s, it is possible to change the stability of the node for testing.
Longer period allows node to have max number of pods (up to 110 pods/node) with
stability.  Shorter period will increase the instability and reduce the number
of pods.

With the periodSeconds to adjust the node stability for testing, it is simpler
to create one test pod and scale the pod for testing.  So, the steps are:

    ./initNamespace.sh
    ./createtestpods.sh 1 1 30000

then scale in/out with:

    ./scale-httpperf.sh <total number of pods>

A deploySimple.yaml is also provided if all you need is a very simple http pod with no services which you create without port number, e.g.

    ./createsimple.sh 1 1

If test requires more services, then you should create each test pod in its own
namespace and services.  Example, 50 test pods in its own namespace will
give you 50 services in total:

    ./initNamespace.sh
    ./cloneNamespace.sh 2 50
    ./createtestpods.sh 1 50 30000

If test requires more endpoints, then you can scale up the httpperf1 pod.
This will increase the number of endpoints for the 3 services in httpperf1.

Example, scale up to 100 pods from 1 pod for httpperf1 will increase the
endpoints from 5 to 500.

    ./scale-httpperf.sh 100

### Start/stop Kubelet

Scripts startKubelet.sh and stopKubelet.sh are provided to use ssh to
start/stop kubelet on all nodes after ssh access is configured to all nodes.

To monitor the nodes, monitornode.sh will send all node status to node.log
repeatedly.

    nohup ./monitornode.sh &

then

    tail -f node.log

Remember to kill monitornode.sh process after test.

To check and ensure all pods are in Ready state, waitPodReady.sh will
repeatedly check pod status until all pods are in Ready state.

    ./waitPodReady.sh

## Cleanup

To clean up cluster and delete test pods/services, run command:

    ./deletePods.sh <pod start number> <pod end number>
