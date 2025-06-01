# Copyright 2016,2022 The Kubernetes Authors All rights reserved.
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

# network performance tests in containers
#
# Based on https://github.com/kubernetes/perf-tests/blob/master/network/benchmarks/netperf/Dockerfile
#
FROM debian:stretch

MAINTAINER Dan McGinnes <mcginnes@uk.ibm.com>

ARG IPERF_VERSION=3.9-1

# install binary and remove cache
RUN apt-get update && apt-get upgrade -y \
    && apt-get install -y curl wget net-tools libsctp1 \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get clean
RUN mkdir -p /tmp

RUN wget https://iperf.fr/download/ubuntu/libiperf0_${IPERF_VERSION}_amd64.deb
RUN wget https://iperf.fr/download/ubuntu/iperf3_${IPERF_VERSION}_amd64.deb
RUN dpkg -i libiperf0_${IPERF_VERSION}_amd64.deb iperf3_${IPERF_VERSION}_amd64.deb
RUN rm libiperf0_${IPERF_VERSION}_amd64.deb iperf3_${IPERF_VERSION}_amd64.deb

COPY k8s-netperf/nptests/nptests /usr/bin

ENTRYPOINT ["nptests"]
