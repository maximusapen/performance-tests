#!/bin/bash 
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * IBM Cloud Kubernetes Service, 5737-D43
# * (C) Copyright IBM Corp. 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# This script will be run by root on cruiser worker so no need to do sudo

echo "******************************************"
echo "Check hostname below is the cruiser worker"
echo "******************************************"
hostname

echo "Check os package"
uname -rv

echo "Get update"
apt-get update

# Comment out until package is known"
#apt-get install <package>
#reboot now

