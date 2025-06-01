# Zero Worker Tests

## Objective

* Measure the time to deploy masters, and the performance of the masters once created.

## Description

* This test creates 10 clusters that each have zero worker nodes. This enables us to measure the time to deploy masters. Once the masters are deployed it runs the APIServer load tests against the 10 masters. In this test the APIServer load tests are run from a standalone jmeter driver running on a 16 core perf client host. This will measure the throughput and response times from the masters.

## Charts

Grafana chart:  `_ClusterCreateZeroWorkers`

* Lower numbers are better for the cluster creation time charts.
* Higher numbers show better performance for the throughput charts.
* Lower numbers are better for the latency charts.

## Details
