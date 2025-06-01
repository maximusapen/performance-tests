# etcd-benchmark

This code became out of date, so to reduce maintenance overheads it has been removed

Etcd-benchmark was an old copy of the official [etcd-benchmark](https://github.com/etcd-io/etcd/tree/master/tools/benchmark) tool for etcd.

The primary aim for the copy was to modify the tool to support generating key/value pairs that mimic those seen in the aramada microservices etcd. The extensions include:

* mvcc-get: Test the get performance of the MVCC storage subsystem of etcd (`etcd-benchmark mvcc get ....`).
* pattern.go: Generates key/values pairs based on a set of patterns. Primarily used to mimic armada microservices key/value pairs (`etcd-benchmark pattern ...`).
* watch_tree.go: Tests the performance of etcd processing watch requests, and the client receiving watch events. A tree of key/value pairs is used so inividual keys and branches can be watched. It tests the sending performance by changing the value of the watched keys with concurrent put requests. 

The current community version has several significant updates. The difficulty in merging the two versions is likely to relate to the way reporting has changed in the newer versions. 
