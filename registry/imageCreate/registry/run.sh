#!/bin/sh
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2023 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

set -e

regParams=
while test "$#" -gt 0; do
	regParams="$regParams $1"
	shift
done

# Set up our command to run docker daemon
set -- dockerd \
	--tls=false \
	--host=unix:///var/run/docker.sock \
	--host=tcp://0.0.0.0:2375 \
	--storage-driver=overlay2 \
	"$@"

# We're running Docker, let's pipe through dind
# (and we'll run dind explicitly with "sh" since its shebang is /bin/bash)
set -- sh "$(which dind)" "$@"

exec "$@" &

sleep 30

cat /etc/hosts
echo "$SPECIAL_DNS_RUN"
./registry $regParams
