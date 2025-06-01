# Online Banking Tests

## Objective

* This benchmark emulates a fictional online bank. For more information see <https://github.ibm.com/ibmperf/olb-java/tree/helm>.

## Description

* The application is a simple Java application that simulates front end application to communicate with backend on-premise database server. The helm chart also contains a sub-chart called stub which simulates backend transaction with set sleep time before it returns the data. This test also uses the Kubernetes pod autoscaler to automatically scale up the number of pods during the test.
* The microservices run in pods in the Cruiser and interact with associated databases.

## Charts

Grafana chart:  `_OnlineBanking`

* Higher numbers show better performance for the throughput charts
* Lower numbers are better for the latency charts.

## Details
