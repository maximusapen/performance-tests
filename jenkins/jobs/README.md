# Jenkins jobs backup for Armada-Performance

## Backup

Jenkins now requires SSO for access to api and backups.  Some information here: https://taas.w3ibm.mybluemix.net/guides/jenkins-sso.md .

To get the tokens for, you just need to log into Jenkins with the ID you want to create an API for, and create one in the user configuration page.  When you are logged in to Jenkins in a browser, clicking your name at the top right will take you to your user page, and selecting Configure from the menu on the left will allow you to generate an API key. You can create as many as you need.  We need to create one for https://alchemy-containers-jenkins.swg-devops.com and one for https://alchemy-testing-jenkins.swg-devops.com.

The tokens are added to vault as:

- `STAGE_GLOBAL_ARMPERF_CONT_JENKINS_TOKEN`: Jenkins api token for `alchemy-containers-jenkins.swg-devops.com`
- `STAGE_GLOBAL_ARMPERF_TEST_JENKINS_TOKEN`: Jenkins api token for `alchemy-testing-jenkins.swg-devops.com`

The Backup-Jenkins job (https://alchemy-containers-jenkins.swg-devops.com/job/Armada-Performance/job/perfClient-setup/job/Backup-Jenkins/) is scheduled to run weekly. It will backup all jobs on both Jenkins server except those under `zArchive` to https://github.ibm.com/alchemy-containers/armada-performance-data/tree/master/jenkins/jobs.  A notification is sent to `#armada-perf-bots` with the backup details. When a job is obsolete it should either be deleted or moved under the existing zArchive folder, and the corresponding backup file in GIT should be deleted.

To allow a branch pushed from Backup-Jenkins job, we need to add AlConBld as collaborators in https://github.ibm.com/alchemy-containers/armada-performance/settings/collaboration.

The Backup-Jenkins job calls `backupJenkins.sh` which go through the Jenkins jobs list in containers_jenkins.json and testing_jenkins.json; retrieve the job contents using the link and save to the specified backup directory relative to jenkins/jobs directory.

To import the project with changes or if project is lost, you can do a HTTP POST to the same link with the xml contents.  To recover a project that is gone, create the project with no config to create the project folder before posting the backup.

## Adding new jobs

Add the new jobs to either either containers_jenkins.json and testing_jenkins.json based on the Jenkins server:

- `containers_jenkins.json`: `alchemy-containers-jenkins.swg-devops.com`
- `testing_jenkins.json`: `alchemy-testing-jenkins.swg-devops.com`

If the new jobs are in a new Jenkins directory, add the directory to backupJenkins.sh so a new GIT directory can be created.

## Slack

Slack for alchemy-containers Jenkins:  #alchemy-toolchain
Useful slack for general Jenkins help:  #taas-jenkins-help
