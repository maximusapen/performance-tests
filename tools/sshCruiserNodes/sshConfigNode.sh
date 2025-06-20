#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright Maximus Apen, 2025 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# Replace sshPublicKey with your ssh public key
echo "sshPublicKey" >> /root/.ssh/authorized_keys
sed -i 's/PermitRootLogin.*/PermitRootLogin yes/g' /etc/ssh/sshd_config; killall -1 sshd
