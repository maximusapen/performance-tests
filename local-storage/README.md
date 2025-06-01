# Armada Local Storage Performance Tests

These tests run a Kubernetes job that uses a simple GO client within a debian based image, to dynamically drive a couple of Linux utilities to measure the performance of local storage within a Kubernetes cluster.

This is based heavily on the <GITHUB_ROOT>/alchemy-containers/persistent-storage test, but runs against the worker node's local disk. It uses the same image as that test, but the parameters passed mean it runs against the local worker storage rather than persistent storage volume.

The utilities are:

* fio : Measures read/write performance
* ioping : Measures io latency

For more information see <GITHUB_ROOT>/alchemy-containers/persistent-storage/README.md
