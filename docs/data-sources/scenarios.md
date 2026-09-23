---
page_title: "pathly_scenarios"
description: |-
  Inventory of the organization scenarios, filterable by tag or by folder.
---

# pathly_scenarios

Lists the scenarios of the organization. Every page is walked: a data source
that stopped at the first one would produce silently incomplete `for_each`
loops.

Two common uses: adopting an existing estate without rewriting everything, and
spotting the scenarios created by hand that escape reviews.

## Example

```terraform
data "pathly_scenarios" "prod" {
  filter_tag = "prod"
}

# What the console holds and the code ignores.
output "scenarios_outside_terraform" {
  value = [
    for s in data.pathly_scenarios.prod.scenarios : s.name
    if !contains([for j in pathly_scenario.journey : j.id], s.id)
  ]
}
```

## Schema

### Optional

- `filter_tag` (String) — Keeps only the scenarios carrying this tag.
- `filter_folder` (String) — Keeps only the scenarios in this folder.

Both filters combine, and apply after the read: they do not reduce the number of
API calls.

### Computed

- `scenarios` (List of Object) — Selected scenarios, each with `id`, `name`,
  `type`, `url`, `enabled`, `interval_sec`, `folder`, `severity`, `tags` and
  `last_status`.

## Required scope

`scenarios:read`. A refusal surfaces as an error: returning an empty list would
destroy a whole estate on the next `apply`, if the output feeds a `for_each`.
