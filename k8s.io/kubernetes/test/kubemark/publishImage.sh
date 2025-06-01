#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright Maximus Apen, 2025 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

set -a
source setenv
set +a

cd $KUBE_ROOT
sudo docker build -t ${KUBEMARK_INIT_TAG} ${KUBEMARK_IMAGE_LOCATION}
sudo docker tag ${KUBEMARK_INIT_TAG} stg.icr.io/armada_performance/kubemark:${KUBE_VERSION}
sudo docker tag ${KUBEMARK_INIT_TAG} stg.icr.io/armada_performance/kubemark:default

sudo ibmcloud cr login
sudo docker push stg.icr.io/armada_performance/kubemark:${KUBE_VERSION}
sudo docker push stg.icr.io/armada_performance/kubemark:default
