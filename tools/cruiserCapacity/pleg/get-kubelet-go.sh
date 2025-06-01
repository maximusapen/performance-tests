#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Script to get the kubelet goroutine stacks
# Author:  Dan McGinnes

curl --key /root/cert/admin-key.pem  --cacert /root/cert/admin.pem --cert /root/cert/admin.pem -k https://localhost:10250/debug/pprof/goroutine?debug=2 
