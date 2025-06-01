# A Docker image for automated testing of persistent storage
FROM golang:1.19.6
LABEL maintainer Richard Schofield <richard_schofield@uk.ibm.com>
RUN apt-get update && apt-get upgrade -y \
    && apt-get install --assume-yes --no-install-recommends \
    curl \
    && apt-get install --assume-yes --no-install-recommends \
    fio \
    && apt-get install --assume-yes --no-install-recommends \
    ioping \
    && apt-get install --assume-yes --no-install-recommends \
    wget \
    && apt-get clean

RUN curl -L -o /tmp/dumb-init.deb \
    https://github.com/Yelp/dumb-init/releases/download/v1.2.0/dumb-init_1.2.0_amd64.deb \
    && dpkg -i /tmp/dumb-init.deb \
    && rm /tmp/dumb-init.deb

# Add metrics dependency
WORKDIR /persistent-storage/metrics/bluemix
COPY metrics/bluemix/monitoring.go metrics/bluemix/send-to-Influx.go ./
RUN go mod init metrics

# Add crypto dependency (used by metrics)
WORKDIR /persistent-storage/tools/crypto/utils
COPY tools/crypto/utils/crypto.go ./
RUN go mod init crypto

# Build the persistent-storage image
WORKDIR /persistent-storage
COPY persistent-storage/imageCreate/persistent-storage/persistent-storage.go ./
RUN go mod init persistent-storage
RUN go mod edit -replace github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix=/persistent-storage/metrics/bluemix
RUN go mod edit -replace github.ibm.com/alchemy-containers/armada-performance/tools/crypto/utils=/persistent-storage/tools/crypto/utils
RUN go mod tidy
RUN go install persistent-storage

# Copy the files required by the peristent-storage image at runtime
COPY persistent-storage/imageCreate/persistent-storage/fiojobfile /tmp/fiojobfile
COPY metrics/bluemix/metrics.toml $GOPATH/src/github.ibm.com/alchemy-containers/armada-performance/metrics/bluemix/metrics.toml

# Create directory for metrics - needs to be created with 777 otherwise Openshift containers cannot write to it.
RUN mkdir -p /performance/metrics
RUN chmod -R 777 /performance

ENTRYPOINT ["dumb-init", "-c", "--"]
