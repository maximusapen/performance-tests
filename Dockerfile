FROM golang:1.19.6

WORKDIR /go/src/github.ibm.com/alchemy-containers/armada-performance/

ADD . /go/src/github.ibm.com/alchemy-containers/armada-performance/
RUN make buildgo
CMD ["/bin/bash"]
