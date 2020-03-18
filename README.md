buildenv
========

A tool for generating environment exports from a YAML file. _Now with vault integration!_

Usage
-----

Given a `variables.yml` file like this:
```yaml
---
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
export SECRET_TEST="It Works" # secret/test
# Environment (stage) Vars:
export ENVIRONMENT="stage"
# Environment (stage) Secrets:
export ANOTHER_SECRET="It Still Works" # secret/test
# Datacenter (ndc_one) Specific Vars:
YET_ANOTHER_SECRET: "secretpassword"
export DC="one"
```

*A Note About Vault:* If you have `secrets` defined in either the global or environment scope, it's a mapping from environment variable to the path in vault. Buildenv uses all the standard vault environment variables to communicate with vault (`VAULT_ADDR` and `VAULT_TOKEN` being the two you're most likely to use.)

Developing
----------

To test with vault, run:

```bash
docker-compose up vault -d
export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="test"
vault write secret/test "value=It Works"
vault write secret/test2 "value=It Still Works"
buildenv -e stage
docker-compose down
```
