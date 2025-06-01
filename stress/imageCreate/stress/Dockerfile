# A convenient Docker image for running Linux stress utility with parameters
FROM ubuntu
LABEL maintainer Richard Schofield <richard_schofield@uk.ibm.com>
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        stress \
    && apt-get clean

# upgrade to fix CVE-2019-6454
RUN apt-get upgrade -y libsystemd0

ADD stress/imageCreate/stress/run_stress.sh /
ENTRYPOINT ["/bin/bash", "-c", "/run_stress.sh ${*}", "--"]
