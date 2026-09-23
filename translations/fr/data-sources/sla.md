> **La version anglaise fait référence.** Cette page traduit
> [`docs/data-sources/sla.md`](../../../docs/data-sources/sla.md), seule version lue par le registre Terraform. En cas
> d'écart, l'anglais prévaut.
---
page_title: "pathly_sla"
description: |-
  Availability objectives with their current measurement.
---

# pathly_sla

Reads every SLA target together with the measurement of its window. A target
never measured keeps its definition and leaves the measurement null: a zero
there would read as a total outage.

```terraform
data "pathly_sla" "breached" {
  filter_state = "breached"
}

output "breached_names" {
  value = [for t in data.pathly_sla.breached.targets : t.name]
}
```

## Schema

### Optional

- `filter_scenario_id` (String)
- `filter_state` (String) — Measurement state, for example `ok` or `breached`.

### Computed

- `targets` (List of Object) — Definition (`id`, `scenario_id`,
  `scenario_name`, `name`, `objective_pct`, `window_days`,
  `exclude_maintenance`, `warn_at_budget_ratio`, `enabled`) flattened with the
  measurement (`state`, `uptime_pct`, `eligible_runs`, `ok_runs`,
  `failed_runs`, `excluded_runs`, `error_budget_runs`,
  `error_budget_used_ratio`, `estimated_downtime_minutes`).

Required scope: `sla:read`.
