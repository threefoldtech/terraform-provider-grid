## Presearch

- Create a registration code from presearch [dashboard](https://nodes.presearch.org/dashboard)
- Only provide restoring keys pair if you want to restore old node or leave empty.

NOTE: restoring keys pair must follow this schema

```bash
    PRESEARCH_BACKUP_PRI_KEY = <<EOF
-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQDQjfuZ3uIGOXUP
Qqpw1K85LV6sZWOAntUnhL73GXTWcwBer06yPI1ush8Vj6tdP94hmUFfWW85vYRU
...
-----END PRIVATE KEY-----
      EOF
      PRESEARCH_BACKUP_PUB_KEY = <<EOF
-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA0I37md7iBjl1D0KqcNSv
OS1erGVjgJ7VJ4S+9xl01nMAXq9OsjyNbrIfFY+rXT/eIZlBX1lvOb2EVJ93o1mz
...
-----END PUBLIC KEY-----
      EOF
```

## Requirments

- 1 CPU
- 1 GB RAM
- 10 GB storage
- public ip
