# armada-performance

## Detect-secret tool

This Repository is guarded by detect-secret tool <https://w3.ibm.com/w3publisher/detect-secrets/developer-tool>.

See <https://ibm.ent.box.com/notes/691042726964> for more details for this GIT

Update baseline command is:

    detect-secrets scan --exclude-files go.sum  --update .secrets.baseline

Only the one who generates/updates the baseline needs to install detect-secrets.

### Pre-commit

For all, install pre-commit in your local environment.  You need python3.  See box note for upgrading python to python3 on mac.

- pip install pre-commit

For each armada-performance clone in your local environment, enable pre-commit hook with command:

- pre-commit install
  - This will add .git/hooks/pre-commit file to the GIT clone.

Alternative, Makefile is now updated with a setup step, install pre-commit with:

- make setup

It is not recommended to disable pre-commit hook once enabled.  But there may be occasions initially when you switch to a local branch without the  pre-commit file

If your commit failed in pre-commit and you see the version in .secrets.baseline has changed, you need to upgrade your pre-commit as follows:

- pre-commit clean
- pre-commit gc
- pre-commit autoupdate

Then

- Revert changes in .secrets.baseline and .pre-commit-config.yaml
- Re-commit your changes including any new changes in .secrets.baseline and .pre-commit-config.yaml

You may be prompted to upgrade your detect-secrets on commit with version changes in .secrets.baseline.
Follow instructions (<https://ibm.biz/detect-secrets-how-to-upgrade>)to upgrade detect-secrets, then you need to upgrade your pre-commit with the pre-commit commands above.

To disable pre-commit hook:

- pre-commit uninstall

Alternatively, merge the master branch to your local branch and keep the pre-commit hook enabled.
