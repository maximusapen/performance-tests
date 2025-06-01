# HTTP Scale: In-Cluster Tests

## Objective

* These tests were used by the Kubernetes community to demonstrate high scalability. The tests create multiple vegeta drivers and send http requests to multiple nginx drivers - all within a single cluster. The aim is to generate a large amount of internal traffic to verify it can handle the load. It is based on this test: <https://github.com/kubernetes/contrib/tree/master/scale-demo>.

## Description

* The vegeta http drivers are distributed in pods across the cluster. The http requests are sent to nginx webservers also distributed in pods across the cluster. An aggregator pod collects the throughput data from all of the vegeta drivers to give a total request rate for the entire cluster.

## Charts

Grafana chart:  `_HTTPScale`

* The charts show the throughput and response time for the throughput aggregated across all vegeta drivers.
* Higher numbers show better performance for the throughput (Requests/s).
* Lower numbers show better performance for the response times.

## Details
