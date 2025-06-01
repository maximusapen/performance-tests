# Image for the JMeter master-controller
FROM jmeter-dist-base
LABEL maintainer Richard Schofield <richard_schofield@uk.ibm.com>

# Install netcat for user interaction with the master-controller pod
RUN apt-get update && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends \
    netcat curl \
    && apt-get clean

COPY jmeter-dist/imageCreate/master/run_test.sh /jmeter/

WORKDIR /jmeter

RUN chmod -R 777 /jmeter

EXPOSE 4444 50000
