# Cluster Creation Tests

## Objective

* These tests record the time to create and delete an Armada cluster including masters and workers.

## Description

* The test records the time spent in each cluster creation creation state (requested, provision_pending, provisioning, waiting for master, deploying and deployed) through to full cluster completion.

## Charts

Grafana chart:  `_Cluster_Create`

* The chart shows the time taken to create a new cluster with 5 workers.
* Lower numbers show better performance.

## Details
