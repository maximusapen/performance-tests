### Patch cruiser worker using ssh daemonset

Goals is to run a shell script on a cruiser worker which can only be accessed
via sshdaemon.

To run the script, you need to use the armada-perf-client for the cluster to
get the cruiser config.  So, you can run on armada perf client machine for the
carrier which created the cruiser.

Here are the steps that I use.

Copy the files in /armada-performance/jenkins/carrier/scripts/patch-cruiser-worker
to your own running directory and "chmod +x *.sh" if necessary

## Create the ssh daemon pods for the cruiser cluster

# Run as ". ./create-sshdaemon.sh <CLUSTER_NAME>"
ktsui@stage-dal09-perf4-client-01:~/sshcruiser$ . ./create-sshdaemon.sh KT-Test1
Usage: ./create-sshdaemon.sh <CLUSTER_NAME>
Removing old kubeConfig artefacts
Getting cluster config
Jan 25 12:12:13.447     API request: https://stage-us-south4.containers.test.cloud.ibm.com:443/v1/clusters/KT-Test1/config/admin : "GetClusterConfig - KT-Test1"
Jan 25 12:12:14.528     API response: 200 OK
Configuration for "KT-Test1" written to kubeConfigAdmin-KT-Test1.zip
Jan 25 12:12:14.528     Total Duration: 1.081s
Archive:  kubeConfigAdmin-KT-Test1.zip
   creating: kubeConfig146451702/
  inflating: kubeConfig146451702/admin-key.pem  
  inflating: kubeConfig146451702/admin.pem  
  inflating: kubeConfig146451702/ca-dal09-KT-Test1.pem  
  inflating: kubeConfig146451702/kube-config-dal09-KT-Test1.yml  
kube-config-dal09-KT-Test1.yml
NAME                  READY     STATUS    RESTARTS   AGE       IP              NODE
ssh-daemonset-bfhww   1/1       Running   0          1h        10.142.127.10   10.142.127.10
ssh-daemonset-c4pfp   1/1       Running   0          1h        10.142.127.26   10.142.127.26

# You will now be in kubeConfig* directory.  Remain in this directory for the next step

You can only have one kubeConfig* directory.  The next time you run for same or another cluster, the kubeConfig* directory and zip files will be deleted at the start of create-sshdaemon.sh

Another alternative to using create-sshdaemon.sh is to set the KUBECONFIG to one of the clusters in /performance/config on performance client machine, then run

- kubectl apply -f ../sshdaemon.yaml
- kubectl get pods -o wide

Then run exec-worker.sh ssh-daemonset-<id> patch.sh as described below.

# Modify patch.sh to run what you need.  The update and reboot steps are commented out for now

# Patch each worker via each ssh-daemonset with "../exec-worker.sh ssh-daemonset-<id> ../patch.sh"

Note you may need to answer "yes" for the first time daemonset access the worker
with authenticity check.

ktsui@stage-dal09-perf4-client-01:~/sshcruiser/kubeConfig146451702$ ../exec-worker.sh ssh-daemonset-c4pfp ../patch.sh
+ echo 'Usage: ./exec-worker.sh < sshdaemon pod name > < shell script name >'
Usage: ./exec-worker.sh < sshdaemon pod name > < shell script name >
+ sshdaemon=ssh-daemonset-c4pfp
+ run_script=../patch.sh
+ echo 'Copy script to sshdaemon'
Copy script to sshdaemon
+ kubectl cp ../patch.sh ssh-daemonset-c4pfp:/tmp/../patch.sh
+ echo 'Copy script from sshdaemon to cruiser worker'
Copy script from sshdaemon to cruiser worker
+ kubectl exec ssh-daemonset-c4pfp -it scp /tmp/../patch.sh root@localhost:/tmp/../patch.sh
The authenticity of host 'localhost (127.0.0.1)' can't be established.
ECDSA key fingerprint is SHA256:ncs2Ji0I77+Fn2hJ1YS0s9sbaB6A4oPKc0Byf5sIqJE.
Are you sure you want to continue connecting (yes/no)? yes
Warning: Permanently added 'localhost' (ECDSA) to the list of known hosts.
Ubuntu 16.04.3 LTS
patch.sh                            100%  707     1.0MB/s   0.7KB/s   00:00    
+ echo 'Run script on cruiser worker'
Run script on cruiser worker
+ kubectl exec ssh-daemonset-c4pfp -it ssh root@localhost /tmp/../patch.sh
Ubuntu 16.04.3 LTS
******************************************
Check hostname below is the cruiser worker
******************************************
stage-dal09-cr1f31c42851dd404b898dbb020a739668-w1.cloud.ibm
Check os package
4.13.0-26-generic #29~16.04.2-Ubuntu SMP Tue Jan 9 22:00:44 UTC 2018
Get update
Hit:1 http://mirrors.service.networklayer.com/ubuntu xenial InRelease
Get:2 http://mirrors.service.networklayer.com/ubuntu xenial-updates InRelease [102 kB]
Get:3 http://mirrors.service.networklayer.com/ubuntu xenial-backports InRelease [102 kB]
Get:4 http://mirrors.service.networklayer.com/ubuntu xenial-security InRelease [102 kB]
Get:5 http://mirrors.service.networklayer.com/ubuntu xenial-updates/main Sources [291 kB]
Get:6 http://mirrors.service.networklayer.com/ubuntu xenial-updates/universe Sources [188 kB]
Get:7 http://mirrors.service.networklayer.com/ubuntu xenial-updates/main amd64 Packages [706 kB]
Get:8 http://mirrors.service.networklayer.com/ubuntu xenial-updates/main Translation-en [294 kB]
Get:9 http://mirrors.service.networklayer.com/ubuntu xenial-updates/restricted amd64 Packages [7,560 B]
Get:10 http://mirrors.service.networklayer.com/ubuntu xenial-updates/universe amd64 Packages [577 kB]
Get:11 http://mirrors.service.networklayer.com/ubuntu xenial-updates/universe Translation-en [233 kB]
Get:12 http://mirrors.service.networklayer.com/ubuntu xenial-updates/multiverse amd64 Packages [16.2 kB]
Get:13 http://mirrors.service.networklayer.com/ubuntu xenial-security/main Sources [107 kB]
Get:14 http://mirrors.service.networklayer.com/ubuntu xenial-security/universe Sources [49.3 kB]
Get:15 http://mirrors.service.networklayer.com/ubuntu xenial-security/main amd64 Packages [431 kB]
Get:16 http://mirrors.service.networklayer.com/ubuntu xenial-security/main Translation-en [188 kB]
Get:17 http://mirrors.service.networklayer.com/ubuntu xenial-security/universe amd64 Packages [199 kB]
Get:18 http://mirrors.service.networklayer.com/ubuntu xenial-security/universe Translation-en [101 kB]
Fetched 3,694 kB in 1s (2,567 kB/s)
Reading package lists...


