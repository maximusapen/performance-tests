# Sysbench Tests

## Objective

* Ensure that the underlying Cruiser workers associated with the test cluster are performing as expected. Any degradation in the worker performance will impact all other performance tests, so we need to validate their performance.

## Description

* The tests run a series of cpu, memory and file io tests on each worker to measure performance.

## Charts

Grafana chart:  `_Sysbench`

* The charts show the time per request for a series of cpu, memory and disk operations with different nubers of test threads.
  * Lower numbers show better performance for the time per request metrics.

## Details
