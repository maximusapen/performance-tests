#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Stop etcd-driver load driver(s). etcd-driver must be configured to watch /test1/end

. etcd-perftest-config

etcdctl put --endpoints=${ETCD_ENDPOINTS} /test1/end true
