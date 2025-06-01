#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

set -a
source setenv
set +a

# Build Kubernetes Release
cd ${KUBE_ROOT}
sudo -E make quick-release
sudo chown -R jenkins:"Domain Users" "${KUBE_ROOT}"
sudo chmod -R 775 "${KUBE_ROOT}"

sudo ${KUBE_ROOT}/build/run.sh make kubemark
sudo chown -R jenkins:"Domain Users" "${KUBE_ROOT}"
sudo chmod -R 775 "${KUBE_ROOT}"

cp ${KUBE_ROOT}/_output/dockerized/bin/linux/amd64/kubemark ${KUBEMARK_IMAGE_LOCATION}
sudo chown -R jenkins:"Domain Users" "${KUBEMARK_IMAGE_LOCATION}"
sudo chmod -R 775 "${KUBEMARK_IMAGE_LOCATION}"
