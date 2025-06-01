#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************
# Setup port forwarding to get to Grafana charts

. privCluster
nohup kubectl --namespace monitoring port-forward svc/grafana 3000:3000&

. iperfClient
nohup kubectl --namespace monitoring port-forward svc/grafana 3001:3000&

. iperfServer
nohup kubectl --namespace monitoring port-forward svc/grafana 3002:3000&
