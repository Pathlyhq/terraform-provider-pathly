# Complete example

A monitoring estate described entirely in code: scenarios, availability
objectives, maintenance windows, an alerting webhook, plus a check on scenarios
created outside Terraform.

```sh
export PATHLY_API_TOKEN="sp_…"
terraform init
terraform plan
terraform apply
```

## Scopes the key needs

This example touches all three resource families. The key must carry:

```
scenarios:read    scenarios:write
alerting:read     alerting:write      (webhook)
maintenance:read  maintenance:write   (maintenance windows)
sla:read          sla:write
```

No organization scope: the check performed when the provider is configured
tolerates being refused.

A missing scope produces an error that names the scope in question, at the point
of the first call that needs it.

## Things to watch

- **The state holds the webhook secret.** It is only returned on creation, so
  Terraform has to keep it. Encrypt the backend, and avoid publishing it as an
  `output`.
- **`terraform destroy` deletes the monitoring.** Destroyed scenarios take their
  run history with them. On a production estate, prefer setting `enabled = false`
  before removing the resource from the code.
- **The `scenarios_outside_terraform` output** lists what the console holds and
  the code ignores. That is the signal that a scenario was created by hand, and
  that it will escape review.
