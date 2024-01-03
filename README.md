buildenv
========

A tool for generating environment exports from a YAML file. Variables can be set in plain test, or by specifying vault key-value (version 2) paths and keys (`kv_secrets`) or the older generic / kv paths (`secrets`) where the key name "value" is assumed.

Usage
-----

Given a `variables.yml` file like this:
```yaml
---
vars:
  GLOBAL: "global"

secrets:
  GENERIC_SECRET: "gen/test"
  KV_SECRET: "old/test"
  KV2_SECRET: "secret/oldstyle"

kv_secrets:
  - path: "secret/test"
    vars:
      KV2_ONE: "one"
      KV2_TWO: "two"
  - path: "old/test"
    vars:
      KV1: "value"
  - path: "gen/test"
    vars:
      KV_GENERIC: "value"

environments:
  stage:
    vars:
      ENVIRONMENT: "stage"

    secrets:
      ANOTHER_SECRET: "secret/oldstyle"

    dcs:
      ndc_one:
        vars:
          DC: "one"
        kv_secrets:
          - path: "old/test"
            vars:
              KV2_THREE: "three"
```

Output would look like this:

```
% buildenv -c -e stage -d ndc_one
# Global Variables
export GLOBAL="global"
export KV2_ONE="1" # Path: secret/test, Key: one
export KV2_TWO="2" # Path: secret/test, Key: two
export KV1="old" # Path: old/test, Key: value
export KV_GENERIC="generic" # Path: gen/test, Key: value
export GENERIC_SECRET="generic" # Path: gen/test, Key: value
export KV_SECRET="old" # Path: old/test, Key: value
export KV2_SECRET="default" # Path: secret/oldstyle, Key: value
# Environment: stage
export ENVIRONMENT="stage"
export ANOTHER_SECRET="default" # Path: secret/oldstyle, Key: value
# Datacenter: ndc_one
export DC="one"
export KV2_THREE="3" # Path: old/test, Key: three
```

*A Note About Vault:* If you have `secrets` or `kv_secrets` defined in either the global or environment scope, it's a mapping from environment variable to the path & key in vault. Buildenv uses all the standard vault environment variables to communicate with vault (`VAULT_ADDR` and `VAULT_TOKEN` being the two you're most likely to use.) You can find the complete list [in the vault client docs](https://pkg.go.dev/github.com/hashicorp/vault-client-go@v0.4.2#WithEnvironment).

Running on Linux or in Docker container
----------

It is recommended to use the flag `-m` when running on linux or docker container with swap enabled.  This will attempt to lock memory and prevent secrets from being written to swap space.  If running on a docker container it may be necessary to add `--cap-add=IPC_LOCK` to the `docker run` command or in the `docker-compose` file to allow this. More info can be found at https://hub.docker.com/_/vault under Memory Locking and 'setcap'.

Developing
----------

To test with vault, run:

```bash
docker-compose up vault -d
export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="test"
vault secrets enable -path gen generic
vault secrets enable -version=1 -path old kv
vault kv put secret/test "one=1" "two=2"
vault kv put secret/oldstyle "value=default"
vault kv put old/test "value=old" "three=3"
vault write gen/test "value=generic"

buildenv -c -e stage -d ndc_one
docker-compose down
```
