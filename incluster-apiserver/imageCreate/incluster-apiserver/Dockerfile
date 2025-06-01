# Docker image to run incluster-apiserver benchmark
FROM alpine
LABEL maintainer Dan McGinnes <mcginnes@uk.ibm.com>

RUN apk --update upgrade

COPY incluster-apiserver/imageCreate/incluster-apiserver/incluster-apiserver /incluster-apiserver
RUN chmod 755 /incluster-apiserver
