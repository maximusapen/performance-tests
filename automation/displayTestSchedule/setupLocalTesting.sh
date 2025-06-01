#!/bin/bash
# ******************************************************************************
# * Licensed Materials - Property of IBM
# * , 5737-D43
# * (C) Copyright IBM Corp. 2023 All Rights Reserved.
# * US Government Users Restricted Rights - Use, duplication or
# * disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
# ******************************************************************************

# This is just a little script to help setup local testing of this code. It assumes that
# it is being run from the armada-performance/automation/displayTestSchedule directory
# and that you also have the armada-performance-data repo cloned to the same git env.

# Need to get to root directory of repo
cd ../..

# Need to copy files from armada-performance-data
cp ../armada-performance-data/automation/assignments ./automation/assignments
cp ../armada-performance-data/automation/client.json ./automation/client.json
cp ../armada-performance-data/automation/schedule.json ./automation/schedule.json

# Run parse/generate script
automation/bin/parseSchedule.sh > /tmp/perfAutomationSchedule.json