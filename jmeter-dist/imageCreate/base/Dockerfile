# Base image for running JMeter based tests
FROM openjdk:8-jre-slim
LABEL maintainer Richard Schofield <richard_schofield@uk.ibm.com>

ARG JMETER_VERSION=5.5
ARG JMETER_PLUGIN_VERSION=2.2

# libfreetype6 libfontconfig1 added to fix runtime issues
RUN apt-get update && apt-get install -y --no-install-recommends \
  wget \
  zip \
  unzip \
  tar \
  libfreetype6 libfontconfig1

RUN mkdir /jmeter

RUN wget https://dlcdn.apache.org//jmeter/binaries/apache-jmeter-$JMETER_VERSION.tgz
RUN wget https://dlcdn.apache.org//jmeter/binaries/apache-jmeter-$JMETER_VERSION.tgz.sha512

RUN cat apache-jmeter-$JMETER_VERSION.tgz.sha512 | sha512sum -c

RUN tar -zxf apache-jmeter-$JMETER_VERSION.tgz -C /jmeter
RUN rm apache-jmeter-$JMETER_VERSION.tgz

WORKDIR /jmeter/apache-jmeter-$JMETER_VERSION
RUN wget https://jmeter-plugins.org/files/packages/jpgc-cmd-$JMETER_PLUGIN_VERSION.zip
RUN wget https://jmeter-plugins.org/files/packages/jpgc-synthesis-$JMETER_PLUGIN_VERSION.zip
RUN wget https://jmeter-plugins.org/files/packages/jpgc-filterresults-$JMETER_PLUGIN_VERSION.zip

RUN unzip jpgc-cmd-$JMETER_PLUGIN_VERSION.zip 
RUN unzip -o jpgc-synthesis-$JMETER_PLUGIN_VERSION.zip
RUN unzip -o jpgc-filterresults-$JMETER_PLUGIN_VERSION.zip

RUN rm jpgc-cmd-$JMETER_PLUGIN_VERSION.zip
RUN rm jpgc-synthesis-$JMETER_PLUGIN_VERSION.zip
RUN rm jpgc-filterresults-$JMETER_PLUGIN_VERSION.zip

RUN echo "log4j2.formatMsgNoLookups=true" >> ./bin/system.properties
RUN zip -q -d ./lib/log4j-core-*.jar org/apache/logging/log4j/core/lookup/JndiLookup.class

RUN chmod -R 777 /jmeter

# Java uses this to store user preferences, so need to make it accesible to nonroot IDs

RUN mkdir -p /etc/.java/.systemPrefs
RUN touch /etc/.java/.systemPrefs/.system.lock
RUN touch /etc/.java/.systemPrefs/.systemRootModFile
RUN chmod -R 755 /etc/.java
RUN mkdir -p /.java/.userPrefs
RUN chmod -R 777 /.java/.userPrefs

ENV PATH /jmeter/apache-jmeter-$JMETER_VERSION/bin:$PATH
