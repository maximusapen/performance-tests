# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Used by pull_fake_cruiser_timing.sh
BEGIN {
    cruiser=""
    print "cruiser,\"total time (sec)\",\"wait for etcd cluster (sec)\",\"wait for master (sec)\", \"wait for 1st etcd backup\",\"wait for 2nd etcd backup\""
}
/fakecruiser/ {
    if (length(cruiser) != 0) {
        print cruiser "," real "," cluster "," master "," backup1 "," backup2
        real=""
        cluster=""
        master=""
        backup1=""
        backup2=""
    }

    cruiser=$0
}
/create-etcd : Wait for the cluster to be healthy/ {
    split($0, a, "--- ")
    cluster=substr(a[2], 1, length(a[2])-1)
    }
/master-service : wait for kubX master to come online/ {
    split($0, a, "--- ")
    master=substr(a[2], 1, length(a[2])-1)
    }
/run-etcd-backup : wait for it to complete/ {
    split($0, a, "--- ")
    if (length(backup1) == 0 ) {
        backup1=substr(a[2], 1, length(a[2])-1)
    } else {
        backup2=substr(a[2], 1, length(a[2])-1)
    }
    }
/real / {
    real=$2
    }
{
    echo "Undexpected line: $0"
}
END {
    if (length(cruiser) != 0) {
        print cruiser "," real "," cluster "," master "," backup1 "," backup2
    }
    }
