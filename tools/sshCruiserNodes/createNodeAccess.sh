#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************


export worker_name=$1

kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: getnodeaccess-${worker_name}
  labels:
    name: getnodeaccess
spec:
  tolerations:
    - operator: "Exists"
  hostNetwork: true
  hostPID: true
  hostIPC: true
  containers:
    - name: getnodeaccess123
      securityContext:
        privileged: true
      image: kitch/sshdaemonset
      volumeMounts:
      - mountPath: /host
        name: host-root
  volumes:
  - name: host-root
    hostPath:
      path: /
  nodeSelector:
    kubernetes.io/hostname: ${worker_name}
EOF

