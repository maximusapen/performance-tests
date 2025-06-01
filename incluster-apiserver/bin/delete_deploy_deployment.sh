#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2021 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# This script is called by run_deployment_test.sh so all deployments
# can be deleted in parallel to create more stress load to apiserver.
ns=$1
helm uninstall "incluster-apiserver-target" --namespace ${ns}
kubectl delete namespace ${ns}
