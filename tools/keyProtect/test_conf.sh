#!/bin/bash -e
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2019, 2020 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

carrier=carrier5
declare -i clusterStart=1
declare -i clusterEnd=950
declare -i secretStart=1
declare -i secretEnd=20

declare -i numThread=50

declare -i numCluster=${clusterEnd}-${clusterStart}+1
declare -i numMpods=${numCluster}*3

# If changing clusterStart, clusterEnd, set numThread so batch is a whole number
declare -i batch=$(((${clusterEnd} - ${clusterStart} + 1) / ${numThread}))
