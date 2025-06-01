#!/bin/bash -x

# Set up your KUBECONFIG before running this script
#export KUBECONFIG=<cruiser kube config file>

workers=$(kubectl get node | grep -v NAME | awk '{print $1}')
echo $workers

if [[ -z $workers ]]; then
    # No workers found
    echo "No workers found"
    echo "Export KUBECONFIG before running this script."
    exit 1
fi

# Check public key is setup in sshConfigNode.sh before starting run
sshPublicKey=$(grep "sshPublicKey" sshConfigNode.sh | grep echo)

if [[ ! -z $sshPublicKey ]]; then
    # Place holder not replaced by ssh public key
    echo "Replace place hold sshPublicKey with your ssh public key"
    echo "before running this script."
    exit 1
fi

for worker in ${workers[@]}
do
    echo Creating node access for $worker
    ./createNodeAccess.sh $worker
done

for worker in ${workers[@]}
do
    echo Setting up ssh for $worker
    # Privileged pod getnodeaccess has a mount in /host to the worker node
    kubectl cp sshConfigNode.sh getnodeaccess-$worker:/host/tmp/sshConfigNode.sh
    # Then use runon to run sshConfigNode.sh on the worker node
    runon $worker sudo /tmp/sshConfigNode.sh
done

# cleanup access nodes
echo Deleting node access pods
./deleteAllNodeAccess.sh

# Cleanup runon jobs/pods
echo Deleting runon jobs
jobs=$(kubectl get job -n ibm-system | grep -v NAME | awk '{print $1}')

for job in $jobs; do
    echo $job
    kubectl delete -n ibm-system job.batch/$job --force --timeout=10s
done
