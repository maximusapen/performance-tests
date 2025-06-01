# Create the run-sysbench image

# use this file to create te docker image - it assumes you are running it from the directory containing the dockerfile.
# substitute the appropriate registry for your environment

# To create and push the image to the registry:
# create tagged image ready to push : docker build -t stg.icr.io/armada_performance_stage1/run-sysbench .
# push image to registry : "docker push stg.icr.io/armada_performance_stage1/run-sysbench"

FROM golang:1.19.6

# Make sure sysbench is installed (This will install latest version)

RUN apt-get update && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends \
    curl \
    apt-transport-https \
    wget \
    && apt-get clean

RUN curl -s https://packagecloud.io/install/repositories/akopytov/sysbench/script.deb.sh | bash
RUN apt -y install sysbench

# Create directory for metrics - needs to be created with 777 otherwise Openshift containers cannot write to it.
RUN mkdir -p /performance/metrics
WORKDIR /performance

COPY sysbench/imageCreate/run-sysbench/my.cnf /etc/mysql/my.cnf

COPY metrics/bluemix/metrics.toml $GOPATH/src/github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix/metrics.toml
COPY metrics/bluemix/alerts.toml $GOPATH/src/github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix/alerts.toml

COPY sysbench/imageCreate/run-sysbench/executeSysbench.sh /performance/executeSysbench.sh
COPY sysbench/imageCreate/run-sysbench/run-sysbench /performance/run-sysbench
COPY sysbench/imageCreate/run-sysbench/run-sysbench /performance/run.sh

RUN chmod -R 777 /performance

ENTRYPOINT ["./run.sh"]
