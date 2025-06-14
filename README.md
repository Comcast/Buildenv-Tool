buildenv
========

A tool for generating environment exports from a YAML file. Variables can be set in plain test, or by specifying vault key-value (version 2) paths and keys (`kv_secrets`) or the older generic / kv paths (`secrets`) where the key name "value" is assumed. Buildenv will autodetect between version 2 and version 1 `kv_secret` paths _unless it can't read the mount details_. For that case, `kv_secrets` will assume version 2, and `kv1_secrets` will use version 1.

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

kv1_secrets:
- path: "old/test"
    vars:
      KV1SPECIFIC: "value"

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

```bash
% buildenv -c -e stage -d ndc_one
# Global Variables
export GLOBAL="global"
export KV2_ONE="1" # Path: secret/test, Key: one
export KV2_TWO="2" # Path: secret/test, Key: two
export KV1="old" # Path: old/test, Key: value
export KV_GENERIC="generic" # Path: gen/test, Key: value
export KV1SPECIFIC="old" # Path: old/test, Key: value
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

Another mode uses -r to run a command.  All exports will be provided directly to a subshell invoked with the command.  This is especially useful in the context of a Makefile where it's very awkward to export lists of environment variables. An added benefit is it's now trivial to set environment variables just for a single command without causing any side-effects for subsequent commands.

Example Makefile:

```
list-buckets: creds.yml
 buildenv -e stage -f $< -r "aws s3 ls"
```

If it's necessary to merge or save a set of variables (for example, so that vault does not need to be called repeatedly), the -u option allows for saving and using a set of variables from the environment without writing possibly sensitive data out to a file:

```bash
% export SAVED_ENV=`echo '{"example_var": "the value"}' | base64`
% buildenv -u SAVED_ENV -f /dev/null
export example_var="the value"
```

This takes a base64 encoded json object with key-value pairs and treats them as additional input variables.  The corresponding flag for export in the same format is -x:

```bash
% buildenv -u SAVED_ENV -f /dev/null -x | base64 -d
{"example_var":"the value"}
```

Multiple -u options can be used as well as combined with -f to combine multiple sources.  Given the above variables.yml:

```bash
% export SAVED_ENV=`echo '{"example_var": "the value"}' | base64`
% export SAVED_ENV2=`echo '{"another_var": "another value"}' | base64`
% buildenv -u SAVED_ENV -u SAVED_ENV2 -v
export GLOBAL="global"
export example_var="the value"
export another_var="another value"
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
