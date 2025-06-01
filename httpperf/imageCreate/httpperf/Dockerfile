# Create the httperf image

# use this file to create te docker image - it assumes you are running it from the directory containing the dockerfile.
# substitute the appropriate registry for your environment

# To create and push the image to the registry:
# create tagged image ready to push : docker build -t stg.icr.io/armada_performance_stage1/httpperf .
# push image to registry : "docker push stg.icr.io/armada_performance_stage1/httpperf"
# This installs a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.

FROM golang:1.19.6

# Install prerequisites
RUN apt-get update && apt-get upgrade -y && apt-get install -y --no-install-recommends \
    curl \
    wget \
    && apt-get clean

CMD /bin/bash

# Build the httpperf command inside the container.
# Build the http-perf image
WORKDIR /httpperf
COPY httpperf/imageCreate/httpperf/server.go ./
RUN go mod init httpperf
RUN go mod tidy

RUN go install httpperf
ADD httpperf/imageCreate/httpperf/server.pem ./
ADD httpperf/imageCreate/httpperf/server.key ./
RUN chmod -R 755 ./*

#Run the httpperf command by default when the container starts.
ENTRYPOINT ["/go/bin/httpperf", "-httpPort=8080", "-httpsPort=8443"]

# Document that the service listens on the ports
EXPOSE 8080
EXPOSE 8443
