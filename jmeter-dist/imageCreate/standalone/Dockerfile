# Image for running the JMeter standalone/worker nodes
FROM jmeter-dist-base
LABEL maintainer Richard Schofield <richard_schofield@uk.ibm.com>

RUN apt-get update && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends \
        netcat \
    && apt-get clean

# TODO check on need for 1099 and 40000 as from origional slave
EXPOSE 4444

COPY jmeter-dist/imageCreate/standalone/run_test.sh /jmeter/

WORKDIR /jmeter

RUN chmod -R 777 /jmeter
