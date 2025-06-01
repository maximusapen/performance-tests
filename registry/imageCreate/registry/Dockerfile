FROM docker:dind
# install ibmcloud
RUN apk update && apk upgrade && apk add bash sudo curl
RUN curl -fsSL https://clis.cloud.ibm.com/install/linux | sh

# install cr plugin and turn off update checking
RUN ibmcloud config --check-version false
RUN ibmcloud plugin install container-registry -r "IBM Cloud"
COPY registry/imageCreate/registry/registry /registry
COPY registry/imageCreate/registry/run.sh /run.sh
COPY registry/imageCreate/registry/artifacts /artifacts

ENV GOPATH /go
COPY metrics/bluemix/metrics.toml $GOPATH/src/github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix/metrics.toml
COPY metrics/bluemix/alerts.toml $GOPATH/src/github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix/alerts.toml

WORKDIR /artifacts
RUN tar -cvf build.tar ./*
WORKDIR /

RUN chmod 755 /run.sh
RUN chmod 755 /registry

ENTRYPOINT ["/run.sh"]
