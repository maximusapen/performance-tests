
# Set up ssh to all cruiser nodes

Follow these instructions to set up ssh to all cruiser nodes with your own ssh key with no pain

## Update script with your SSH public key

Replace placeholder sshPublicKey in sshConfigNode.sh with your SSH public key.

## Run script to set up SSH to all cruiser nodes with your SSH public key

You need to have the runon script available on your system.  Get runon from https://github.ibm.com/kubernetes-tools/runon.  There is a modified version of runon included here which allows you to use runon concurrently.  The main change is adding the random JOB_NAME to ${TEMP_DIR}/job.yml as ${TEMP_DIR}/job-${JOB_NAME}.yml and a couple of wait that is required for clusters as big as 100 nodes.

Export KUBECONFIG to configure access to your cruiser cluster.

Run sshSetupCluster.sh to configure ssh access to all nodes using your SSH public key.
Script will also clean up all the pods and jobs created for the configuration.

This may take a long time to run if you have a big cluster.  Run with nohup and check nohup.out for any errors:

  nohup ./sshSetupCluster.sh &

You can ignore these errors:
- Unable to use a TTY - input is not a terminal or the right kind of file
- E0117 17:11:26.138730   85106 v2.go:105] read /dev/stdin: bad file descriptor

## Check ssh is setup for all nodes and allows initial ssh setup like updating known_hosts file

Run
  ./checkKubelet.sh

As cruiser node IPs are reused, your ssh known_hosts file may have an entry in known_hosts
file from previous clusters.  If you are getting the remote host identification warning
as below, you can remove known_hosts file and rerun checkKubelet.sh.

The script attempts ssh to each node as root to run pwd.  The return value expected is /root.
ssh -o StrictHostKeyChecking=no root@$node pwd

@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
IT IS POSSIBLE THAT SOMEONE IS DOING SOMETHING NASTY!
Someone could be eavesdropping on you right now (man-in-the-middle attack)!
It is also possible that a host key has just been changed.
The fingerprint for the ECDSA key sent by the remote host is
SHA256:BV2SyntDA1Aq9KxzwWDGYcVe8VKpxKmMLYX2gA+BGwA.
Please contact your system administrator.
Add correct host key in /home/SSO/ktsui/.ssh/known_hosts to get rid of this message.
Offending ECDSA key in /home/SSO/ktsui/.ssh/known_hosts:196
  remove with:
  ssh-keygen -f "/home/SSO/ktsui/.ssh/known_hosts" -R 10.143.201.97
Password authentication is disabled to avoid man-in-the-middle attacks.
Keyboard-interactive authentication is disabled to avoid man-in-the-middle attacks.

## Verify

You should now be able ssh to any of the cruiser nodes with your own .ssh setup
  ssh root@<cruiser node IP>

## Remove root access ASAP to minimize security risk

Run disableRootLogin.sh script.
