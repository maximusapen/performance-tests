# lease_test

This is a simple program that generates, and sustains, a load of 250,000 etcd leases for as long as the program runs. 250,000 leases was chosen because that is what the Dallas armada microservices etcd was seen to have in Dec 2020. The loadbalancer endpoint for etcd is currently hardcoded. 
