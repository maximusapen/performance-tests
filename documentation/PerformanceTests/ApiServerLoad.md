# APIServer Load Tests

## Objective

* Measure the performance of the IKS master apiservers.

## Description

* This test creates 10 clusters that each have zero worker nodes. Once the masters are deployed it runs the APIServer load tests against the 10 masters. In this test the APIServer load tests are run using a distributed load driver, using 11 driver nodes by default (1 master & 10 slaves).  This will measure the throughput and response times from the masters.
* The load driver makes GET pods & GET namespaces requests against the master apiservers.

## Charts

Grafana chart:  ``

* Lower numbers are better for the cluster creation time charts.
* Higher numbers show better performance for the throughput charts.
* Lower numbers are better for the latency charts.

## Details
