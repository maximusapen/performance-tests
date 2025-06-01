#!/usr/bin/env bash

# Copyright 2018, 2023 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Cloud information
RANDGEN=$(dd if=/dev/urandom bs=64 count=1 2>/dev/null | base64 | tr -d "=+/" | dd bs=16 count=1 2>/dev/null | sed 's/[A-Z]//g')
# shellcheck disable=2034 # Variable sourced in other scripts.
KUBE_NAMESPACE="kubemark_${RANDGEN}"
KUBEMARK_IMAGE_TAG="${KUBEMARK_IMAGE_TAG:-latest}"
KUBEMARK_IMAGE_LOCATION="${KUBEMARK_IMAGE_LOCATION:-${KUBE_ROOT}/cluster/images/kubemark}"
KUBEMARK_INIT_TAG="${KUBEMARK_INIT_TAG:-${PROJECT}:${KUBEMARK_IMAGE_TAG}}"
CLUSTER_LOCATION="${CLUSTER_LOCATION:-dal09}"
REGISTRY_LOGIN_URL="${REGISTRY_LOGIN_URL:-https://test.cloud.ibm.com}"

# Default Log level for all components in test clusters and variables to override it in specific components.
TEST_CLUSTER_LOG_LEVEL="${TEST_CLUSTER_LOG_LEVEL:---v=4}"

HOLLOW_KUBELET_TEST_LOG_LEVEL="${HOLLOW_KUBELET_TEST_LOG_LEVEL:-$TEST_CLUSTER_LOG_LEVEL}"
HOLLOW_PROXY_TEST_LOG_LEVEL="${HOLLOW_PROXY_TEST_LOG_LEVEL:-$TEST_CLUSTER_LOG_LEVEL}"

# Hollow-node components' test arguments.
HOLLOW_KUBELET_TEST_ARGS="${HOLLOW_KUBELET_TEST_ARGS:-} ${HOLLOW_KUBELET_TEST_LOG_LEVEL}"
HOLLOW_PROXY_TEST_ARGS="${HOLLOW_PROXY_TEST_ARGS:-} ${HOLLOW_PROXY_TEST_LOG_LEVEL}"

# User defined
# number of real workers in spawnTester cluster
NUM_NODES="${NUM_NODES:-2}"
# spec of real workers in spawnTester cluster
NODE_SIZE=${NODE_SIZE:-u2c.2x4}
DESIRED_NODES="${DESIRED_NODES:-10}"
# number of hollow nodes
# TODO: once IKS supports `kubetest`, i.e. the base cluster provisioning implemented in `kubetest`
# we can use NUM_NODES=${KUBEMARK_NUM_NODES:-10} to avoid usage of KUBEMARK_NUM_NODES
# more context, see https://github.com/kubernetes/kubernetes/pull/76909#discussion_r277735942
KUBEMARK_NUM_NODES="${KUBEMARK_NUM_NODES:-10}"
ENABLE_KUBEMARK_CLUSTER_AUTOSCALER="${ENABLE_KUBEMARK_CLUSTER_AUTOSCALER:-true}"
ENABLE_KUBEMARK_KUBE_DNS="${ENABLE_KUBEMARK_KUBE_DNS:-false}"
KUBELET_TEST_LOG_LEVEL="${KUBELET_TEST_LOG_LEVEL:-"--v=2"}"
KUBEPROXY_TEST_LOG_LEVEL="${KUBEPROXY_TEST_LOG_LEVEL:-"--v=4"}"
USE_REAL_PROXIER=${USE_REAL_PROXIER:-false}
NODE_INSTANCE_PREFIX=${NODE_INSTANCE_PREFIX:-node}
USE_EXISTING=${USE_EXISTING:-}
