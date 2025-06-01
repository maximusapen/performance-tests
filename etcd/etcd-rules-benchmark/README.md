# etcd-rules-benchmark
The scripts in here assume the etcd-rules-benchmark code will have been copied into the performance directory from https://github.ibm.com/alchemy-containers/armada-ballast/tree/master/performance either manually or by using Build-And-Copy-Perf-Repo job.

The `runPerfBenchmarks.sh` script will run different combinations of concurrency settings and send the results to our metrics service.