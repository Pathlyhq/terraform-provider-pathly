---
page_title: "pathly_webhook"
description: |-
  Signed outbound webhook, called when a scenario fails and when it recovers.
---

# pathly_webhook

Outbound webhook. Every delivery is HMAC-signed with the secret returned at
creation time: verify that signature on the receiving side, otherwise anyone can
forge a fake alert.

## Example

```terraform
resource "pathly_webhook" "alerts" {
  url    = "https://hooks.example.com/pathly"
  events = ["run.failed", "run.recovered"]
}
```

## Schema

### Required

- `url` (String, sensitive) — Destination, over https and public. **Forces
  replacement.** Private addresses, internal domains and cloud metadata IPs are
  refused: the worker calls from the platform network, and a webhook pointing at
  `169.254.169.254` would make it read its own credentials.

### Optional

- `events` (List of String) — `run.failed` and `run.recovered`. Both by default.
  **Forces replacement.**

### Computed

- `id` (String) — Identifier assigned by Pathly.
- `enabled` (Boolean) — A webhook disabled from the console stays in the state,
  without being called.
- `url_fingerprint` (String) — Fingerprint of the destination. It changes if the
  URL was modified elsewhere: that is the only way to see it, since the API
  never returns the stored URL.
- `secret` (String, sensitive) — HMAC signing secret, returned **once only** at
  creation time.
- `created_at` (String) — Creation date.

## The secret and the state file

Since the secret is only returned at creation time, Terraform has to keep it: it
is therefore in the state. Three consequences to deal with:

1. **Encrypt the backend** (`azurerm` with encryption at rest, `gcs` with CMEK,
   or an equivalent). A plaintext state in a public bucket exposes the secret.
2. **Do not publish it as a non-sensitive `output`**: it would show up in the CI
   logs.
3. **To move it into a vault**, read it once then remove the output:

```terraform
output "webhook_secret" {
  value     = pathly_webhook.alerts.secret
  sensitive = true
}
```

```sh
terraform output -raw webhook_secret | az keyvault secret set --vault-name … --name pathly-webhook --value @-
```

## Import

```sh
terraform import pathly_webhook.alerts wh_01H8ZK…
```

~> The import emits a warning: the API returns neither the URL nor the secret.
State `url` again in the configuration, otherwise the plan would propose a
replacement, and fetch the secret back from your vault, since it was only
displayed at creation time.
