# buildenv

A tool for generating environment exports from a YAML file. _Now with vault integration!_

# Usage

Given a `variables.yml` file like this:
```yaml
---
# global settings
## example `./buildenv`
  vars:
    GLOBAL: "global"
# vault kv value stores version 2
  secrets_v2:
# multiple values for one key
    KV_V2_TEST:
      path: "kv/data/test/"
      vars:
        KV_TEST_VALUE_1: "value1"
        KV_TEST_VALUE_2: "value2"
        KV_TEST_VALUE_3: "value3"
# single value for one key
    KV_TEST_VALUE_1_SIMPLE:
      path: "kv/data/test/value1"
# vault kv store version 1
  secrets:
    SECRET_TEST_1: "secret/test/value"
    SECRET_TEST_2: "secret/test2/value"
# vault secret engines
  engines:
    - path: "totp/keys/user_test"
      args:
        generate: "true"
        issuer: "Vault"
        account_name: "user@test.com"
      vars:
        BARCODE: "barcode"
  # environment specific settings
  ## example `./buildenv -e prod`
  environments:
    prod:
      vars:
        ENVIRONMENT: "stage"

      secrets:
        ANOTHER_SECRET: "secret/test2/value"
      # environment datacenter specific settings
      ## example `./buildenv -e prod -d aws`
      dcs:
        aws:
          engines:
            - path: "totp/keys/user_aws_test"
              args:
                generate: "true"
                issuer: "Vault"
                account_name: "user_aws@test.com"
              vars:
                AWS_BARCODE: "barcode"
          secrets:
            AWS_SECRET_TEST_1: "secret/test/value"
            AWS_SECRET_TEST_2: "secret/test2/value"
          secrets_v2:
            AWS_KV_TEST:
              path: "kv/data/test/"
              vars:
                AWS_KV_TEST_VALUE_1: "value1"
                AWS_KV_TEST_VALUE_2: "value2"
                AWS_KV_TEST_VALUE_3: "value3"
          vars:
            CLOUD: "aws"
        ## example `./buildenv -e prod -d hetzner`
        hetzner:
          vars:
            CLOUD: "hetzner"
```

# Example

Output would look like this:

```
$ buildenv -e prod -d hetzner
# Setting Variables for:
# Environment: prod
# Datacenter: hetzner
# Global Vars:
export GLOBAL="global"
# Global Secrets:
export SECRET_TEST_1="It Works" # secret/test/value
export SECRET_TEST_2="It Still Works" # secret/test2/value
# Global Secrets V2:
export KV_V2_TEST_SIMPLE=It Still Works # kv/data/test value1
export KV_TEST_VALUE_3=true # kv/data/test/ value3
export KV_TEST_VALUE_1=It Still Works # kv/data/test/ value1
export KV_TEST_VALUE_2=It Still Works # kv/data/test/ value2
# Global Engine:
export BARCODE=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 # totp/keys/user_test map["account_name":"user@test.com" "generate":"true" "issuer":"Vault"]
# Environment (prod) Vars:
export ENVIRONMENT="stage"
# Environment (prod) Secrets:
export ANOTHER_SECRET="It Still Works" # secret/test2/value
# Environment (prod) Secrets V2:
# Environment (prod) Engines:
# Datacenter (prod) Specific Vars:
export CLOUD="hetzner"
# Datacenter (prod) Specific Secrets:
# Datacenter (prod) Specific Secrets V2:
# Datacenter (prod) Specific Engine:
```

*A Note About Vault:* If you have `secrets` defined in either the global or environment scope, it's a mapping from environment variable to the path in vault. Buildenv uses all the standard vault environment variables to communicate with vault (`VAULT_ADDR` and `VAULT_TOKEN` being the two you're most likely to use.)

# Running on Linux or in Docker container

It is recommended to use the flag `-m` when running on linux or docker container with swap enabled.  This will attempt to lock memory and prevent secrets from being written to swap space.  If running on a docker container it may be necessary to add `--cap-add=IPC_LOCK` to the `docker run` command or in the `docker-compose` file to allow this. More info can be found at https://hub.docker.com/_/vault under Memory Locking and 'setcap'.

# Developing

To test with vault, run:

```bash
docker-compose up vault -d
export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="test"

vault secrets enable -version=1 --path secret kv
vault secrets enable -version=2 kv
vault secrets enable totp
# enable vault audit logging to stdout for easier debugging
vault audit enable file file_path=stdout

vault kv put secret/test value="It Works"
vault kv put secret/test2 value="It Still Works"

vault kv put kv/test value1="It Still Works" value2="It Still Works" value3=true

buildenv -e prod -d aws
#How set the environment variables from buildenv command
eval "$(buildenv -e prod -d aws)"
docker-compose down
```
