# Cluster Autoscaler Tests

## Test Objective

The aim of the Cluster Autoscaler tests is to evaluate the performance of the Autoscaler pod, which is responsible for automatically adding and deleting cluster workers based on application resource requests. Details of the Autoscaler configuration can be found here: [Cluster Autoscaler](https://console.test.cloud.ibm.com/docs/containers/cs_cluster_scaling.html#ca).

## How the Test Works
The test performs the following operations:

Initialisation
- Creates a new worker pool called asPool
- Installs and enables the autoscaler to work with asPool
- Creates a deployment called scaler which is an nginx app requesting 1.1 cores
- Waits for the new worker pool to be Ready with the initial specified number of scaler pod replicas running. 
    - Only one scaler pod, requiring one worker, is requested by default. The initial number of replicas, and hence initial nodes ordered, can be overidden in the RunAuto.sh script using the autoscaler_initial_testpod_replicas parameter if required

Scale up
- Once the initial configuarion is running, starts the metrics and scales the scaler app to the requested number of replicas. This will cause the autoscaler to order more workers as required until the scaler resource requests are satisfied.
- When all nodes are ready, and therefore all scaler pods are running, the test is complete.
- Sends the time taken to scale up pods and nodes to grafana.

Scale down
- Starts the scale down by setting the requested replicas to 1. 
- Records the time taken for the runnning pods to scale to 1 (a few seconds) and the time taken for all the extra worker nodes ordered by the autoscaler to be deleted
- When only two nodes remain, sends the time taken to scale down pods and nodes to grafana.


Clean up
- Deletes the scaler application, the autoscaler and the asPool worker pool.

## Running the Autoscaler Tests

The Node-Autoscaler tests are run from the Armada-Performance/Automation/Run-Performance-Tests Jenkins job. The Autoscaler introduces two new parameters:

- NODE_AUTOSCALER_TESTPOD_REPLICAS: This parameter specifies the number of scaler test pods replicas that will be created. Each test pod requests 1.1 cpus. So the number of worker nodes ordered will depend on the number of cpus of the cluster workers in the asPool. Currently the machine type is set to u2c.2x4 in RunAuto.sh:  autoscaler_machine_type parameter). For 2x4 nodes, only 1 pod will fit per node. So in that case the parameter represents the number of nodes the autoscaler will order.

- NODE_AUTOSCALER_MAX_NODES: This parameter specifies the maximum number of workers that the autoscaler can create.
