#!/bin/sh
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2017, 2018 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

set -e

sysbenchParams=
while test "$#" -gt 0; do
  regParams="$sysbenchParams $1"
  shift
done

echo "sysbenchParams are:" $sysbenchParams

./run-sysbench $sysbenchParams
