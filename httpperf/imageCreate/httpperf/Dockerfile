
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
