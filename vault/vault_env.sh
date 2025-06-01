#!/bin/bash
#

# Set up common vault environment

VAULT_ADDR=https://vserv-us.sos.ibm.com:8200
VAULT_PATH=generic/crn/v1/staging/public/containers-kubernetes/us-south/-/-/-/-/stage/armada-performance

# Disable writing to SECRET_FILE.  Only enable in dire straits.
# SECRET_FILE=$HOME/.ssh/armada_performance_id

# Other envs that are not secret
armada_performance_account_id=4a160c3a25d49f6171b796555191f7da
armada_performance_prod_account_id=641f32b9227848fdd5d2ab94f1ab4343
armada_performance_functional_id=armada.performance@uk.ibm.com
PROD_GLOBAL_ARMPERF_SOFTLAYER_1186049_USERID=1186049_armada.performance@uk.ibm.com
PROD_GLOBAL_ARMPERF_SOFTLAYER_1540207_USERID=IBM1540207
prod_carrierussouth_api_server_ip=origin.us-south.containers.cloud.ibm.com
prod_carrierussouth_iks_endpoint=https://origin.us-south.containers.cloud.ibm.com
satellite0_api_server_ip=containers.test.cloud.ibm.com
satellite0_iks_endpoint=https://containers.test.cloud.ibm.com
stage_carrier4_api_server_ip=stage-us-south4.containers.test.cloud.ibm.com
stage_carrier4_iks_endpoint=https://stage-us-south4.containers.test.cloud.ibm.com
stage_carrier5_api_server_ip=stage-us-south5.containers.test.cloud.ibm.com
stage_carrier5_iks_endpoint=https://stage-us-south5.containers.test.cloud.ibm.com
