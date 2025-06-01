# Image for running the JMeter slave/worker nodes
FROM jmeter-dist-base
LABEL maintainer Richard Schofield <richard_schofield@uk.ibm.com>

EXPOSE 1099 40000

RUN apt-get update && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends \
    && apt-get clean

WORKDIR /jmeter

# Dummy file for request body contents
RUN touch .json

# Dummy hosts file
RUN touch hosts.csv

RUN chmod -R 777 /jmeter

ENTRYPOINT jmeter-server -Dhttpclient.reset_state_on_thread_group_iteration=false -Dserver.rmi.ssl.disable=true -Dserver.rmi.localport=40000 -Dserver_port=1099 -Djavax.net.ssl.keyStore=/jmeter/cert.jks -Djavax.net.ssl.keyStorePassword="${KEYSTORE_PWD}" -Dhttpclient4.time_to_live=3600000 -f -l samples.out
