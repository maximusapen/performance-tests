# AcmeAir

On Performance client, you will find 3 blueperf acmeair directories under /performance:

- src/blueperf :
  
  This is the location where scripts will actually run from - so if making changes manually make sure you change these ones.
  - helper - Runtime files used by runAuto.sh in Run-Performance-Tests Jenkins job
  - acmeair-driver/acmeair-jmeter/scripts - check jmeter results here:
    - acmeair.log
    - output*.csv
    - results*.jtl
- armada-perf/blueperf
  
  Symbolic link to the below location
- src/github.ibm.com/alchemy-containers/armada-performance/blueperf
  
  This is the version of code in our git repo - but this is not executed at runtime, this code is copied to `src/blueperf` by BACPR, and that is the location that is used at runtime.

## AcmeAir image

When adding new acmeair test variations, you may need to build the image.  Building Image is included when testing with Jenkins job `Acmeair-image`.

## Test connections

curl -v http://<ingress_or_route_host_name>/booking/loader/load
curl -v http://<ingress_or_route_host_name>/booking
curl -v http://<ingress_or_route_host_name>/auth
curl -v http://<ingress_or_route_host_name>/flight
curl -v http://<ingress_or_route_host_name>/customer
