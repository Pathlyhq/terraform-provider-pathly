---
page_title: "Pathly Provider"
description: |-
  Manage Pathly monitoring as code: scenarios, maintenance windows, webhooks and SLA targets.
---

# Pathly Provider

Describe Pathly monitoring like the rest of your infrastructure: reviewed in a
*pull request*, applied by CI, identical from one environment to the next.

```terraform
terraform {
  required_providers {
    pathly = {
      source  = "pathlyhq/pathly"
      version = "~> 0.1"
    }
  }
}

provider "pathly" {}

resource "pathly_scenario" "checkout" {
  name         = "Checkout"
  url          = "https://shop.example.com/cart"
  interval_sec = 300
  expect_text  = "Your cart"
  severity     = "critical"
}
```

## Authentication

```sh
export PATHLY_API_TOKEN="sp_…"
```

The key is created under **Settings → API keys**, from an owner or
administrator account. Give it the strictly necessary scopes:

| Managed resources | Scopes |
|---|---|
| `pathly_scenario`, `pathly_scenarios` | `scenarios:read`, `scenarios:write` |
| `pathly_webhook` | `alerting:read`, `alerting:write` |
| `pathly_maintenance_window` | `maintenance:read`, `maintenance:write` |
| `pathly_sla_target` | `sla:read`, `sla:write` |

For a pipeline that only runs `terraform plan`, the `:read` scopes are enough.
No organization scope is required: the check made at configuration time
tolerates its refusal.

## Schema

### Optional

- `api_token` (String, sensitive) — Organization API key, `sp_` prefix.
  Better supplied through `PATHLY_API_TOKEN`: written into a `.tf` file it ends
  up in the repository, and read from a Terraform variable it ends up in
  plaintext in the state file.
- `api_url` (String) — API base. Defaults to `https://api.pathlyhq.com`.
  Plaintext HTTP is accepted on `localhost` only: anywhere else, the key would
  travel readable in the `Authorization` header.

The `PATHLY_API_TOKEN` and `PATHLY_API_URL` environment variables take
precedence over the attributes of the block.

## Behavior

- **Configuration**: a verification call is made once. An expired, revoked or
  truncated key fails here, with a message that says what to fix, instead of one
  error per resource during the plan. A key that is valid but has limited scopes
  passes: a refusal of that call is not an error.
- **Rate limit**: the `Retry-After` header is honored, with a cap of 90 seconds
  and four attempts.
- **Creations**: every creation carries an idempotency key. If the response is
  lost, the retry returns the resource already created rather than a second
  copy.
- **Vanished resource**: it leaves the state and the next plan recreates it. A
  refusal or an outage is never mistaken for a disappearance.

## Out of scope, deliberately

The provider manages no organizations, no users, no API keys, no notification
recipients and no incident muting. The first three would turn a token leak into
a takeover of the organization, the fourth would put personal data into a state
file, and the fifth would mean an `apply` wakes up a scenario that was
deliberately muted.
