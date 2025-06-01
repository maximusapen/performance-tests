# Carrier-lock
A utility to control a lock at the carrier level. This is intended to be used to coordinate tests that hit the masters hard as we do not want multiple tests running at the same time.

It uses a ConfigMap created on the carrier to operate the lock.

## Usage
Under normal operation the flow should be:

```
carrier-lock .... --action acquire
<Execute test that hits the carrier master hard>
carrier-lock .... --action release
```
THe `cleanup` function is used periodically by this job -> https://alchemy-testing-jenkins.swg-devops.com/job/Armada-performance/job/Automation/job/Check-Orphaned-Carrier-Lock/build?delay=0sec to test if there are any locks that were taken by processes that are no longer running.

### acquire
Use this to acquire the lock and stop other users from acquiring it. If the lock is already owned then wait for the `--max-wait-time` parameter duration. 
e.g.

```
carrier-lock --kubeconfig /performance/config/carrier4_stgiks/admin-kubeconfig --max-wait-time 120m --action acquire
```

### release
Use this to release the lock once finished using it, and other clients can start. This will only succeed if called from the same hostname & PID that acquied the lock
e.g.

```
carrier-lock --kubeconfig /performance/config/carrier4_stgiks/admin-kubeconfig --action release
```

### force-release
Same as release, but it will not check for matching hostname & PID.
e.g.

```
carrier-lock --kubeconfig /performance/config/carrier4_stgiks/admin-kubeconfig --action force-release
```

### query
Use this to query the lock status. It will return a json representation of the lock, which contains the host it was taken from, the pid of the owner and the time the lock was taken.
e.g.

```
./carrier-lock --kubeconfig /performance/config/carrier4_stgiks/admin-kubeconfig --action query
{"host":"stgiks-dal10-perf4-client-03","pid":"43552","start-time":"2021-11-11 11:35:06.500703278 +0000 UTC m=+30.321635200"}
```

### cleanup
Use this to release any locks that might have been left behind. It checks that the lock was created from this host, and then checks if the PID that claimed the lock is still running. If the PID is no longer running then it releases the lock.
e.g.

```
carrier-lock --kubeconfig /performance/config/carrier4_stgiks/admin-kubeconfig --action cleanup
```