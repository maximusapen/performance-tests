# Network Tests

Network tests include netperf.

## Objective

* These tests measure Kubernetes network performance and are built from the Kubernetes network performance benchmark. See <https://github.com/kubernetes/perf-tests/tree/master/network/benchmarks/netperf>

## Description

* The tests use containers as workers, each with iperf and other tools built in. Workers 1 and 2 are placed on one node and Worker 3 is placed on a separate node. This allows the following network paths to be tested:
  * Same VM : Worker 1 sending to Worker 2 using PodIP or Cluster IP
  * Remote VM : Worker 3 sending to Worker 2 using PodIP or cluster IP
  * ame VM Pod HairPin : Worker sends to itself using Cluster IP

## Charts

Grafana chart:  `_Netperf`

* The charts show the throughput in Gbits/s
  * Throughput using TCP
  * Throughput using UDP
  * Throughput using the netperf benchmark running TCP tests
* Higher numbers show better performance

## Details
