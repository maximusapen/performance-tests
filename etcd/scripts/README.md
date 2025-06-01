# Etcd scripts
These scripts were created to manage an etcd cluster in Kubernetes where NFS mounts were used for the etcd database. They haven't been used for years.

This set of script uses NFS for the persistent volume. The NFS configuration would have to be update before using.
```
createEtcdCluster.sh
deleteEtcdCluster.sh
etcd-cluster.yml
persistent-volumes.yml
getEndpoints.sh
```

This set of script uses Softlayer NFS for the persistent volume. The NFS configuration would have to be update before using.
```
createEtcdSLNFSSingle.sh
createEtcdSLNFSCluster.sh
deleteEtcdSLNFSCluster.sh
deleteEtcdSLNFSSingle.sh
etcd-slnfs-cluster.yml
etcd-slnfs-single.yml
sl-persistent-volumes.yml
getServiceEndpoints.sh
```

Turns on authentication within the etcd cluster.
```
etcdAddAuthentication.sh <endpoints>
```
