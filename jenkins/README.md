# Jenkins job for Armada-Performance

## Import / Export

The xml files in https://github.ibm.com/alchemy-containers/armada-performance/tree/master/jenkins/jobs are exported from the Jenkins projects in https://alchemy-containers-jenkins.swg-devops.com/job/Armada-Performance/ and https://alchemy-testing-jenkins.swg-devops.com/job/Armada-Performance/.

To export a project, go to the project page and append "/config.xml" to the link, then save to a file.

As an example, Run-Performance-Tests.xml was exported from
https://alchemy-testing-jenkins.swg-devops.com/job/Armada-Performance/job/Automation/job/Run-Performance-Tests/config.xml

Firefox may have a problem viewing the xml 1.1 config files.  Use Safari if you get Exception with Firefox.

To import the project with changes or if project is lost, you can do a HTTP POST to the same link with the xml contents.  To recover a project that is gone, create the project with no config to create the project folder before posting the backup.

## Project API

The import/export link is described in the project api page (append "/api" to project link), e.g.
https://alchemy-testing-jenkins.swg-devops.com/job/Armada-Performance/job/Automation/job/Run-Performance-Tests/api

## Backup

Weekly backup is scheduled in https://alchemy-containers-jenkins.swg-devops.com/job/Armada-Performance/job/perfClient-setup/job/Backup-Jenkins/ which backup all performance Jenkins jobs including Backup-Jenkins itself.

The containers-jenkins-job uses JJB which develops job in git and push to Jenkins.  This is an alternative approach.
See details in [README](https://github.ibm.com/alchemy-containers/containers-jenkins-jobs/blob/master/README.md).

## Carrier Updates

Performance carriers are scheduled to get updated daily or weekly by Jenkins jobs in <https://alchemy-containers-jenkins.swg-devops.com/job/Armada-Performance/job/carrier-pipeline/>

Scripts supporting carrier updates are in jenkins/carrier/launchdarkly directory.

Tugboats are included in schedule updates when updating main carrier:

- Openshift tugboats: stage-dal10-carrier<main_carrier>00
- ETCD tugboats: stage-dal10-carrier<main_carrier>01

Main carriers and tugboats have different micro-services deployed based on the boms in https://github.ibm.com/alchemy-containers/armada-secure/tree/master/boms.  The boms used for the carrier/tugboat is configured in cluster_updater_config.yaml for the carrier/tugboat in armada-secure.

## Slack

Slack for alchemy-containers Jenkins:  #alchemy-toolchain
Useful slack for general Jenkins help:  #taas-jenkins-help
