# K8s-netperf Benchmark Tests

These tests are based on the benchmarks at https://github.com/kubernetes/perf-tests/blob/master/network/benchmarks/netperf/ but have some modifications to run in the armada-performance automation.

They run the following iperf/netperf tests:

```
1 iperf TCP. Same VM using Pod IP
2 iperf TCP. Same VM using Virtual IP
3 iperf TCP. Remote VM using Pod IP
4 iperf TCP. Remote VM using Virtual IP
5 iperf TCP. Hairpin Pod to own Virtual IP
6 iperf UDP. Same VM using Pod IP
7 iperf UDP. Same VM using Virtual IP
8 iperf UDP. Remote VM using Pod IP
9 iperf UDP. Remote VM using Virtual IP
10 netperf. Same VM using Pod IP
11 netperf. Same VM using Virtual IP
12 netperf. Remote VM using Pod IP
13 netperf. Remote VM using Virtual IP
```
