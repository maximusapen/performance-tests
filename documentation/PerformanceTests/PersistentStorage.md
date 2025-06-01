# Persistent Storage Tests

## Objective

* These tests run a Kubernetes job that uses a simple GO client within a debian based image, to dynamically drive a couple of Linux utilities to measure the performance of persistent storage within a Kubernetes cluster.

## Description

* The Kubernetes job runs the following tests with block and file persistent storage
  * fio : Measures read/write performance
  * ioping for classic and vpc : Measures io latency

## Charts

Grafana chart:  `_PersistentStorage`

* Higher numbers show better performance for the IOPS and Bandwidth charts
* Lower numbers are better for the latency charts.

## Details
