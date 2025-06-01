# Vault as a Service for Performance

To run the scripts in vault, you need to set up Vault CLI.  See `Setting up CLI` in Box note:
https://ibm.ent.box.com/notes/428142255914 for install Vault and getting GIT personal access token.

For new secrets added to vault, you need to add the new key to vault_keylist
for Jenkins jobs.

Armada-Performance squad members can write, list and delete vault keys but cannot read the key values for the following Vault environment which is set up in `vault_env.sh`:
```
VAULT_ADDR=https://vserv-us.sos.ibm.com:8200
VAULT_PATH=generic/crn/v1/staging/public/containers-kubernetes/us-south/-/-/-/-/stage/armada-performance
```

## Squad member action

Allowed for Armada-Performance squad members.

### Login Vault as a Service

Login in to vault:
    ./vault_login.sh < your GIT personal access token >

Note, for the GIT personal access token you just need to allow repo access.

After logged in successfuly, you can run the scripts allowed for squad members.

### List Vault keys

To list all vault keys in VAULT_PATH:
    ./vault_list.sh

### Add or Modify Vault secret

To add key/value secret to Vault:
    ./vault_write.sh < key > < value >

Add the new key to vault_keylist file for reading by AppRole.  If key is carrier-specific, add key as stage_carrier< carrier number >_< key >

Add fake secrets for the new key to test_secrets file with data:
< key >=VALUE_< key >

### Delete Vault secret

To delete a key/value secret from Vault:

    ./vault_delete.sh < key >

Delete the key from vault_keylist file.

Remove fake secret from test_secrets file

### Get all Vault secret

    ./vault_getsecret < vault_role_id_value > < vault_secret_id_value >

All secrets are written to $HOME/.ssh/armada_performance_id file.


Note, squad member do not have permissions to read secrets - this script can only be used by jenkins jobs which have the `armada-performance-read` approle,

### Generate performance toml file

For testing purpose using fake secrets in test_secrets (does not require vault login or appRole login):
    ./generate_perf_toml.sh test_secrets

To generate performance toml file using real secrets, run following 2 commands:
        ./vault_getsecret < vault_role_id_value > < vault_secret_id_value >
        ./generate_perf_toml.sh

The generated toml files will be in /tmp/stage_carrier*.toml

## Add new toml files with new secrets

Add new toml template file to template directory.  Generated carrier-specific toml filename will be `stage_carrier< number >_< toml template filename>`, e.g. `stage_carrier*_perf.toml` will be generated from `perf.toml`.

Template files should use the following values for different types of vault secrets so it will be replaced by real secrets:

- common secret: "${< secret key >}"    -- pragma: allowlist secret
- carrier-specific secret: "CARRIER_< secret key >  -- pragma: allowlist secret

If there are new secrets for the new toml, follow step `Add Vault secret` above.

Now you can test the toml generation by running:
    ./generate_perf_toml.sh test_secrets
