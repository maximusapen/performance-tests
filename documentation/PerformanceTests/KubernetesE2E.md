# Kubernetes End to End Tests

## Objective

* These tests measure kubernetes performance and are built from the Kubernetes community E2E tests. See <https://github.com/kubernetes/perf-tests/tree/master/clusterloader2/>

## Description

* Kubernetes load performance tests: This test creates 30 pods per node using replication controllers. These tests do not have secrets, configmaps or daemons. The workload consists of several replication controllers with a mixture of 5 or 30 pods each.
* The test records the time to create, delete and scale pods and services.
* The test also includes api response times for: {put, post, list, get and delete} operating on {pods, nodes, services, deployments and replication controllers}

## Charts

Grafana chart:  `_K8s_E2e_Tests`

* The charts show the API response time to:
  * put, post, list, get and delete `Pods`
  * get and list `Nodes`
  * post, list, get and delete `Services`
  * get `Deployments`
  * post, list, get and delete `Secrets`
* A chart also shows the time taken to complete each phase of the test including the initial creation of 30 pods per node, the scaling of those pods up and down (twice) and the deletion of those pods.

## Details
