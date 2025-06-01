# Network Related Tools

## Traffic modification using `tc` (Traffic Control) 
`tc` can be used to disrupt network traffic (https://www.dasblinkenlichten.com/simulating-latency-and-packet-loss-on-a-linux-host/). Within IKS there is a need to disrupt traffic across many workers for a certain period of time. For example, drop 50% of the packets going through eth0/1 interfaces on all dal13 workers, for a period of 30 minutes. This can be accomplished on a worker with the following commands. 
```
sudo tc qdisc add dev eth0 root netem loss 50%
sudo tc qdisc add dev eth1 root netem loss 50%
sleep 1800
sudo tc qdisc del dev eth0 root netem loss 50%
sudo tc qdisc del dev eth1 root netem loss 50%
```

The trick is running those commands on all the desired workers. This is done with a daemonset. In the sample the nodes are selected via a node selector:
```
      nodeSelector:
          failure-domain.beta.kubernetes.io/zone: dal13
```

The command are separated into 3 initContainers, which run sequentially (i.e. setup, sleep, revert). The daemonset is such that if a particular worker has reached the pod limit, kube will remove an existing pod to make    room for the tc-apply pod.
* tc-apply.yml - Definition of the daemonset. 
* applyTcApply.sh - Updates the tc-apply.yml so that kube will see that it changes, applies tc-apply.yml and waits till it completes so that the end time is documented.

