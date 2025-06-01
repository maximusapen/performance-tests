# Acmeair driver

- Follow the instructions here (https://github.com/blueperf/acmeair-driver), but use the AcmeAir-microservices-mpJwt.jmx script.

- Also set the jmeter property (in jmeter.properties): CookieManager.save.cookies=true

Note.  AcmeAir-microservices-mpJwt.jmx in armada-performance GIT is a copy of https://github.com/blueperf/acmeair-mainservice-java/blob/master/jmeter-files/AcmeAir-microservices-mpJwt.jmx which uses uses JWT auth in HTTP header required for master branch.  Other branches like `microprofile-2.0` works with https://github.com/blueperf/acmeair-driver/blob/master/acmeair-jmeter/scripts/AcmeAir-microservices.jmx which is cloned to perf clients with Build-and-Copy-perf-repo Jenkins job.
