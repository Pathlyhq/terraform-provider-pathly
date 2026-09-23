---
page_title: "pathly_maintenance_windows"
description: |-
  Maintenance windows, recurring and one-off.
---

# pathly_maintenance_windows

Lists the windows. Recurring and one-off windows share the same shape: the
attributes of the other kind stay null.

```terraform
data "pathly_maintenance_windows" "checkout" {
  filter_scenario_id = pathly_scenario.checkout.id
}
```

## Schema

### Optional

- `filter_scenario_id` (String)

### Computed

- `windows` (List of Object) — `id`, `scenario_id`, `starts_at`, `ends_at`,
  `reason`, `weekday`, `start_minute`, `duration_min`.

Required scope: `maintenance:read`.
