---
page_title: "pathly_sla_target"
description: |-
  SLA target measured over a rolling window, with an error budget.
---

# pathly_sla_target

An SLA target, measured over a rolling window with an error budget.

## Example

```terraform
resource "pathly_sla_target" "checkout" {
  scenario_id          = pathly_scenario.checkout.id
  name                 = "Checkout funnel — 99.9 %"
  objective_pct        = 99.9
  window_days          = 30
  exclude_maintenance  = true
  warn_at_budget_ratio = 0.8
}
```

With these values, the target tolerates about 43 minutes of unavailability per
30-day period, and warns when 80% of that budget has been consumed, which is
around 35 minutes, while there is still room to act.

## Schema

### Required

- `objective_pct` (Number) — Availability aimed for, from 50 to 100. For example
  `99.9`.
- `window_days` (Number) — Rolling measurement window, in days, from 1 to 365.

### Optional

- `scenario_id` (String) — Measured scenario. When absent, the target applies to
  the whole organization. **Forces replacement**: a target is identified by its
  scenario, moving it would amount to measuring something else under the same
  identifier.
- `name` (String) — Label displayed in reports. 120 characters at most.
- `exclude_maintenance` (Boolean) — When true, maintenance windows do not
  consume the error budget.
- `warn_at_budget_ratio` (Number) — Share of the error budget consumed that
  triggers the warning, from 0.1 to 1.
- `enabled` (Boolean) — When false, the target stays defined but no longer
  warns.

### Computed

- `id` (String) — Identifier assigned by Pathly.

## Creation and modification

The API exposes an *upsert*: creation and modification go through the same call,
identified by the targeted scenario. A target already defined in the console for
that scenario will therefore be adopted by the first `apply`, without a conflict
error.

## Import

```sh
terraform import pathly_sla_target.checkout sla_01H8ZK…
```
