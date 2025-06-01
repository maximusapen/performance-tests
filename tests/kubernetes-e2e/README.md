# Kubernetes E2E Testing

## Scalability

### Density

Density tests perform the following steps:
1. Create a single Kubernetes resource (i.e. ReplicationController, Deployment or Job). This resource is configured to create the appropriate number of pods / node in the cluster. The startup times for these pods are measured.
2. Once these pods are running, an additional pod is scheduled on each node to measure startup latencies.
3. The additional pods are removed and the time to delete and terminate them is measured.

##### Feature: Performance
* should allow starting 30 pods per node using { ReplicationController} with 0 secrets, 0 configmaps and 0 daemons

##### Feature: Manual Performance
* should allow starting 3 pods per node using { ReplicationController} with 0 secrets, 0 configmaps and 0 daemons  

* should allow starting 30 pods per node using { ReplicationController} with 0 secrets, 0 configmaps and 2 daemons  

* should allow starting 50 pods per node using { ReplicationController} with 0 secrets, 0 configmaps and 0 daemons  

* should allow starting 100 pods per node using { ReplicationController} with 0 secrets, 0 configmaps and 0 daemons  

* should allow starting 30 pods per node using {extensions Deployment} with 0 secrets, 0 configmaps and 0 daemons  

* should be able to handle 30 pods per node {extensions Deployment} with 2 secrets, 0 configmaps and 0 daemons  

* should allow starting 30 pods per node using {extensions Deployment} with 0 secrets, 2 configmaps and 0 daemons  

* should allow starting 30 pods per node using {batch Job} with 0 secrets, 0 configmaps and 0 daemons  


##### Feature: High Density Performance
* should allow starting 95 pods per node using { ReplicationController} with 0 secrets, 0 configmaps and 0 daemons

### Load

Load tests perform the following steps:
1. A number of Kubernetes resources are created and configured to create the appropriate number of pods / node in the cluster.
The resources are configured to be of three different sizes:
 * Small: 5 Pods (small resources own 50% of total pods)
 * Medium: 30 Pods (medium resources own 25% of total pods)
 * Big: 250 Pods (big resources own 25% of total pods)  

2. Scales Resource for first time (pods associated with each resource may be increased or decreased)  

3. Scales Resource for second time (pods associated with each resource may be increased or decreased)

4. Resources are deleted and the time to delete and terminate them is measured.  

##### Feature: Performance
* should be able to handle 30 pods per node { ReplicationController} with 0 secrets, 0 configmaps and 0 daemons

##### Feature: Manual Performance
* should be able to handle 3 pods per node { ReplicationController} with 0 secrets, 0 configmaps and 0 daemons  

* should be able to handle 30 pods per node { ReplicationController} with 0 secrets, 0 configmaps and 2 daemons  

* should be able to handle 30 pods per node {extensions Deployment} with 0 secrets, 0 configmaps and 0 daemons  

* should be able to handle 30 pods per node {extensions Deployment} with 2 secrets, 0 configmaps and 0 daemons  

* should be able to handle 30 pods per node {extensions Deployment} with 0 secrets, 2 configmaps and 0 daemons  

* should be able to handle 30 pods per node { Random} with 0 secrets, 0 configmaps and 0 daemons  

* should be able to handle 30 pods per node {batch Job} with 0 secrets, 0 configmaps and 0 daemons

### Empty
##### Feature: Empty
* starts a pod
