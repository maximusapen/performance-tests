# Build Kubemark image

## Setup Environment

Update setenv KUBE_VERSION to the kube version for the image you want to build

Check the GO version to use for the build.

To override the GO version used in the kubernetes code base, modify these files on a perf client:

/performance/src/k8s.io/kubernetes/build/root/WORKSPACE

- Update `go_version =` with the new GO version

/performance/src/k8s.io/kubernetes/test/images/Makefile

- Update `GOLANG_VERSION=` with the new GO version

/performance/src/k8s.io/kubernetes/build/build-image/cross/Dockerfile

- Update `FROM golang:` with the new GO version

/performance/src/k8s.io/kubernetes/build/build-image/cross/VERSION

- Replace `v<GO_version>-1` with the new GO version

## Build and Publish image

To build and publish kubemark image, Run these scripts on a perf client.

Use `nohup < command > &` due to  the unreliable vpn and check nohup.out.

    nohup ./buildImage.sh &
    nohup ./publishImage.sh &
