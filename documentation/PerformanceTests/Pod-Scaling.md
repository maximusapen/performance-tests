# Pod Scaling Tests

## Objective

* These tests measure the time to scale up/down the number of replicas.

## Description

* The tests starts with a single pod and scales up using `kubectl scale --replicas=xxx`. Pods are considered to be started when they are in 'Running' state. The pods are then scaled back to one pod.

## Charts

Grafana chart:  `_PodScaling`

* The charts show the time taken to scale from 1 to 250 replicas and from 1 to 500 replicas and the time to scale back to 1 replica.
  * Lower numbers show better performance.

## Details
