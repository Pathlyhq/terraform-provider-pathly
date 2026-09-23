---
page_title: "pathly_sla_targets"
description: |-
  Availability objectives, definitions only.
---

# pathly_sla_targets

Lists the SLA definitions without their measurement. Use [`pathly_sla`](sla.md)
when the current score is what you need, and the `pathly_sla_target` resource
to change a definition.

```terraform
data "pathly_sla_targets" "all" {}
```

## Schema

### Computed

- `targets` (List of Object) — `id`, `scenario_id`, `name`, `objective_pct`,
  `window_days`, `exclude_maintenance`, `warn_at_budget_ratio`, `enabled`.

Required scope: `sla:read`.
