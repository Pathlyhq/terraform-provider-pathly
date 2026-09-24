---
page_title: "pathly_maintenance_window"
description: |-
  Maintenance window, one-off or weekly. Failures that happen inside it do not trigger any alert.
---

# pathly_maintenance_window

A window during which failures do not alert and, if the SLA targets ask for it,
do not consume the error budget.

Two forms, mutually exclusive:

- **weekly** — `weekday`, `start_minute` and `duration_min`,
- **one-off** — `starts_at` and `ends_at`.

Without `scenario_id`, the window covers the whole organization.

~> **No in-place modification.** The API cannot rewrite a window: any change
destroys the resource and recreates it. This is visible in the plan.

## Examples

```terraform
# Backup, every Sunday from 3 am to 5 am.
resource "pathly_maintenance_window" "backup" {
  weekday      = 7
  start_minute = 180
  duration_min = 120
  reason       = "Weekly backup"
}

# Announced migration, on a single scenario.
resource "pathly_maintenance_window" "migration" {
  scenario_id = pathly_scenario.checkout.id
  starts_at   = "2026-10-04T22:00:00Z"
  ends_at     = "2026-10-05T02:00:00Z"
  reason      = "Checkout funnel migration"
}
```

## Schema

### Optional

Every attribute **forces replacement**.

- `scenario_id` (String) — Targeted scenario. When absent, the window covers the
  whole organization.
- `starts_at` (String) — Start, in ISO 8601. One-off form.
- `ends_at` (String) — End, in ISO 8601. One-off form. The API refuses a one-off
  window longer than 90 days: a window that drags on hides real outages.
- `weekday` (Number) — Day of the week, 1 for Monday through 7 for Sunday.
  Weekly form.
- `start_minute` (Number) — Start minute within the day, from 0 to 1439 (UTC
  time). `180` means 3 am.
- `duration_min` (Number) — Duration in minutes, from 15 to 1440.
- `reason` (String) — Reason displayed in the console and in the audit trail.
  500 characters at most.

### Computed

- `id` (String) — Identifier assigned by Pathly.

## Import

Bring an existing window into state. Prefer an `import` block (Terraform /
OpenTofu ≥ 1.5); the CLI form is equivalent.

```terraform
import {
  to = pathly_maintenance_window.backup
  id = "mw_01H8ZK…"
}
```

```sh
terraform import pathly_maintenance_window.backup mw_01H8ZK…
```

Full working example: [`examples/import`](../../examples/import/).
