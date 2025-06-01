# Identify stranded IAM Service IDs

A bash script that will retrieve a list of IAM Service IDs in an account, and identify those service IDs that do not have an active IKS cluster bound to them. If the -delete flag is passed it will delete them, otherwise it will just list the stranded ones.

Before executing the script you must first ensure that the perf-metadata.toml files for all carriers, are available locally in the \<armada-perf-repo-root\>/armada-perf-client2/config folder. The instructions here -> https://github.ibm.com/alchemy-containers/armada-performance/tree/master/vault will tell you how to generate these, but by far the easiest way is to run this script on a perf client.

The even easier option is to use this Jenkins job to run it -> https://alchemy-testing-jenkins.swg-devops.com/view/Armada-performance/job/Armada-Performance/job/Automation/job/IdentifyStrandedIAMServiceIds/

To run:  
```
./identifyStrandedIamIds.sh
```
Or to have it actually delete any stranded IDs it finds:
```
./identifyStrandedIamIds.sh -delete
````
