---
page_title: "pathly_settings"
description: |-
  Organization-wide settings: alerting, status page, timezone and triage.
---

# pathly_settings

Organization-wide settings: alerting, escalation, status page, timezone and
triage.

There is exactly one settings object per organization. Declaring this resource
twice would have the two definitions overwrite each other on every apply.
Create and Update are the same call: the settings exist before Terraform.

`terraform destroy` does **not** reset them. There is no deletion endpoint, and
there should not be one: destroying a test workspace would silently cut the
alerting of the whole organization. Terraform only stops tracking them.

## Simple example

```terraform
resource "pathly_settings" "this" {
  timezone    = "Europe/Paris"
  alert_email = "ops@example.com"
}
```

## Complex example

```terraform
resource "pathly_settings" "this" {
  timezone               = "Europe/Paris"
  alert_email            = "ops@example.com"
  alert_on_recovery      = true
  escalation_after_fails = 3
  escalation_email       = "oncall@example.com"
  weekly_digest_enabled  = true
  weekly_digest_email    = "ops@example.com"

  status_slug   = "acme"
  status_public = true

  ssl_warn_days    = [30, 14, 7]
  domain_warn_days = [30]

  triage_enabled         = true
  triage_confirm_enabled = true
  triage_latency_factor  = 2.5
}
```

Turning `status_public` on publishes the scenario names and their availability.
Check that none of them names an internal host before you do.

Attributes left out of the configuration keep the value already set in the
console. Sending a zero would erase them on the first apply.

## Schema

### Optional

- `alert_email` (String) — Address that receives the failure alerts.
- `alert_on_recovery` (Boolean) — When true, a recovery is announced as well.
- `escalation_after_fails` (Number) — Consecutive failures before escalating,
  1 to 100.
- `escalation_email` (String) — Address warned once the threshold is reached.
- `weekly_digest_email` (String) — Recipient of the weekly summary.
- `weekly_digest_enabled` (Boolean) — Sends the weekly summary.
- `status_slug` (String) — Slug of the public status page, 2 to 64 characters,
  lowercase letters, digits and hyphens. Changing it breaks the previous link.
- `status_public` (Boolean) — Exposes the status page to anyone holding the
  link.
- `timezone` (String) — IANA timezone of the reports and the maintenance
  windows.
- `ssl_warn_days` (List of Number) — Days before a certificate expires at
  which to warn, for example `[30, 14, 7]`.
- `domain_warn_days` (List of Number) — Days before a domain expires at which
  to warn.
- `triage_enabled` (Boolean) — Sorts failures between real incidents and
  noise.
- `triage_confirm_enabled` (Boolean) — Runs a confirmation execution before
  opening an incident.
- `triage_latency_factor` (Number) — Multiple of the usual latency above which
  a run is considered degraded, 1 to 100.

### Computed

- `id` (String) — Always `settings`.
- `name` (String) — Organization name.
- `plan_id` (String) — Active plan.
- `has_slack_webhook`, `has_teams_webhook`, `has_discord_webhook`,
  `has_pagerduty`, `has_opsgenie`, `has_datadog`, `has_sentry` (Boolean) —
  A destination is configured. The URL and the key are never returned.

## Import

Bring the organization settings into state. Prefer an `import` block
(Terraform / OpenTofu ≥ 1.5); the CLI form is equivalent.

```terraform
import {
  to = pathly_settings.this
  id = "settings"
}
```

```sh
terraform import pathly_settings.this settings
```

The identifier is ignored: there is only one settings object. Full working
example: [`examples/import`](../../examples/import/).

## Required scopes

`org:read` to refresh, `org:write` to change.
