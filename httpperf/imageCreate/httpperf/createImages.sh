#!/bin/bash

# Generates multiple images (numContainers) all listening on different ports.
# The images are pushed to the imageNames (imageName0, imageName1 etc), registry and namespace defined below.

if [ -z "$1" ]; then
   echo
   echo "Usage: $0 numContainerImages"
   echo
   exit 1
fi
numContainers=$1

registry="stg.icr.io"
nameSpace="armada_performance_stage1"
testName="httpperf"
numContainers=$1
declare -i httpPort=8080
declare -i httpsPort=8443
echo

#cycle through containers
for i in $(seq 0 $(($numContainers - 1))); do
   rm Dockerfile_multi
   sed -e "s/_httpPort_/$((httpPort))/g" -e "s/_httpsPort_/$((httpsPort))/g" Dockerfile.template.txt >Dockerfile_multi
   echo
   echo building and pushing $registry/$nameSpace/$testName$i
   docker build -f Dockerfile_multi -t $registry/$nameSpace/$testName$i .
   docker push $registry/$nameSpace/$testName$i
   ((httpPort++))
   ((httpsPort++))
done

echo
echo Finished!
