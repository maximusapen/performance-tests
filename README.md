
## Detect-Secret Tool

This repository is protected by the [detect-secrets tool](https://w3.ibm.com/w3publisher/detect-secrets/developer-tool).

For further details about this repository, refer to the [Box note](https://ibm.ent.box.com/notes/691042726964).

To update the secrets baseline, run:

  detect-secrets scan --exclude-files go.sum --update .secrets.baseline

Only those responsible for generating or updating the baseline need to install detect-secrets.

### Pre-commit

All contributors should install pre-commit in their local environment. Python 3 is required. Refer to the Box note for instructions on upgrading to Python 3 on macOS.

- `pip install pre-commit`

For each local clone of armada-performance, enable the pre-commit hook with:

- `pre-commit install`
  - This command adds the `.git/hooks/pre-commit` file to your repository.

Alternatively, you can use the Makefile to install pre-commit:

- `make setup`

Disabling the pre-commit hook is not recommended. However, if you switch to a local branch that does not have the pre-commit configuration, you may need to temporarily disable it.

If your commit fails due to pre-commit and you notice changes in `.secrets.baseline`, upgrade pre-commit as follows:

- `pre-commit clean`
- `pre-commit gc`
- `pre-commit autoupdate`

Then:

- Revert changes in `.secrets.baseline` and `.pre-commit-config.yaml`
- Re-commit your changes, including any updates to `.secrets.baseline` and `.pre-commit-config.yaml`

If prompted to upgrade detect-secrets due to version changes in `.secrets.baseline`, follow the [upgrade instructions](https://ibm.biz/detect-secrets-how-to-upgrade), then update pre-commit using the commands above.

To disable the pre-commit hook:

- `pre-commit uninstall`

Alternatively, merge the master branch into your local branch and keep the pre-commit hook enabled.
