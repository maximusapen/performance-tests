# A convenient Docker image for running iperf with parameters
FROM ubuntu:18.04
LABEL maintainer Katherine Tsui <ktsui@uk.ibm.com>
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        iperf3 \
    && apt-get clean

# upgrade to fix CVE-2019-6454
RUN apt-get upgrade -y libsystemd0
ADD iperf/imageCreate/iperf/run_iperf.sh /
RUN chmod 755 /run_iperf.sh
ENTRYPOINT ["/bin/bash", "-c", "/run_iperf.sh ${*}", "--"]
