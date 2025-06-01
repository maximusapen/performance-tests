# Sysbench Benchmark Tests

These tests run the sysbench benchmark tool from https://github.com/akopytov/sysbench

The tests are run by deploying a kubernetes daemonset which creates a pod on each worker node in a cruiser. The image used by the pod runs a go program which installs sysbench then runs a set of benchmark tests (cpu, memory and file io ) via the executesysbench.sh script and then sends the results to the metrics service.
