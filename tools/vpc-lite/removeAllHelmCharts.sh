#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Helper script to delete all helm charts. Update cluster names before using.

. privCluster
helm uninstall jmeter-dist --namespace httpperf
helm uninstall jmeter-standalone --namespace httpperf
helm uninstall httpperf --namespace httpperf

. iperfClient
helm uninstall jmeter-dist --namespace httpperf
helm uninstall jmeter-standalone --namespace httpperf
helm uninstall httpperf --namespace httpperf

. iperfServer
helm uninstall jmeter-dist --namespace httpperf
helm uninstall jmeter-standalone --namespace httpperf
helm uninstall httpperf --namespace httpperf
