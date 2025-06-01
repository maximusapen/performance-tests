# Logspeed utility

These tests can be used to measure the speed of obtaining logs from a kubernetes pod.

### log-gen.yaml
This is a daemonset that will create a container that generates logs on each node. The logs are generated in an Init Container - so the pod will only go `Running` once the logs have finished generating.

The size of logs generated is controlled by this code:
```
while [ "$i" -le 100000 ]; do
    str=$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 500 | head -n 1);
```
The number in the while loop is how many lines will be generated, and `fold -w 500` is how many bytes in each line.

The total log size should be `lines x (bytes per line)` .

Note in testing it was found that if we set the total sie to 100MB the logs could get culled, so would not be the expected size. 50MB logs seemed to work well.

### Running the tests

The `run_logspeed.sh` script can be used to execute the tests. It will install the daemonset on a cluster, wait for the pods to be running, and then gather the logs from each pod and time how long it takes, and provide metrics on how long it took, and the rate of log transfer.

**Usage:**
```
./run_logspeed.sh <repeats>
```
`repeats` is how many times it will loop over getting the logs from each pod

e.g.
```
./run_logspeed.sh 5
```

**Results:**
The script will print the result from each pod:
```
13:12:08 - Got logs from pod logger-lmg54 on node 10.143.193.79 in 9 seconds. File size was 51588895 . Log download rate was 5732099 Bytes per second 
13:12:18 - Got logs from pod logger-vv8nb on node 10.143.159.126 in 10 seconds. File size was 51588895 . Log download rate was 5158889 Bytes per second
```

And then print a summary at the end:

```
13:12:27 - Test completed - Min time: 7, Mean time: 8, Max time: 11, Overall transfer rate: 5916157 bytes per second
```


