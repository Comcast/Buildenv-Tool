buildenv
========

A tool for generating environment exports from a YAML file. _Now with vault integration!_

Usage
-----

Given a `variables.yml` file like this:
```yaml
---
  vars:
    GLOBAL: "global"

  secrets:
    SECRET_TEST: "secret/test"

  environments:
    stage:
      vars:
        ENVIRONMENT: "stage"

      secrets:
        ANOTHER_SECRET: "secret/test2"

      dcs:
        ndc_one:
          secrets:
            YET_ANOTHER_SECRET: "secret/test3"
          vars:
            DC: "one"
          kvsecrets:
            ODD_KEY_SECRET:
              path: "secret/test4"
              key: "stage"

        ndc_two:
          secrets:
            YET_ANOTHER_SECRET: "secret/test3"
          vars:
            DC: "one"
```

Output would look like this:

```
% buildenv -e stage -d ndc_one
# Setting Variables for:
# Environment: stage
# Datacenter: ndc_one
# Global Vars:
export GLOBAL="global"
# Global Secrets:
export SECRET_TEST="It Works" # secret/test1
# Environment (stage) Vars:
export ENVIRONMENT="stage"
# Environment (stage) Secrets:
export ANOTHER_SECRET="It Still Works" # secret/test2
# Datacenter (ndc_one) Specific Vars:
YET_ANOTHER_SECRET: "secretpassword"
export DC="one"
# KV Secrets:
export ODD_KEY_SECRET="It Still Works" # secret/test3
```

*A Note About Vault:* If you have `secrets` defined in either the global or environment scope, it's a mapping from environment variable to the path in vault. Buildenv uses all the standard vault environment variables to communicate with vault (`VAULT_ADDR` and `VAULT_TOKEN` being the two you're most likely to use.)

*A Note About Keys:* Vault secrets defined under `secrets:` must have the key `value`.  Vault secrets defined under `kvsecrets:` can have any key as long as it's defined under the corresponding `key:`.

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
vault write secret/test "value=It Works"
vault write secret/test2 "value=It Still Works"
vault write secret/testKeyPath1 "key1=Odd Key Value1 Works"
vault write secret/testKeyPath3 "key3=Odd Key Value3 Works"
vault write secret/testKeyPath5 "key5=Odd Key Value5 Still Works"
vault write secret/testKeyPath7 "key7=Odd Key Value7 Still Works"
buildenv -e stage
docker-compose down
```
