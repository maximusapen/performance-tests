# HTTP Tests

HTTP Tests include http-perf and https-perf.

## Objective

* These tests evaluate the http and https performance of containers using Nodeports, LoadBalancers, LoadBalancer2 and Ingress.

## Description

* The client is jmeter running in distributed mode, with 11 driver nodes by default (1 master & 10 slaves) which sends http and https requests with different numbers of threads. All request sizes are a few bytes but the response payload can be a few bytes or 1KB (50% of the time for each case).
* The server is a GO http server running in a container. One server is deployed to each Cruiser node. The incoming requests are distributed across the containers using NodePorts, a LoadBalancer or Ingress.

## Charts

Grafana charts:  `_HTTP`, `_HTTPS`

* The charts show the throughput, response time and % error responses with different numbers of jmeter threads. There is a set of charts for Nodeports, LoadBalancers and Ingress using http and https.
  * Higher numbers show better performance for the throughput (Requests/s).
  * Lower numbers show better performance for the response times.
  * Zero is the expected result for '%Request error' charts.

## Details
