> **La version anglaise fait référence.** Cette page traduit
> [`docs/data-sources/runs.md`](../../../docs/data-sources/runs.md), seule version lue par le registre Terraform. En cas
> d'écart, l'anglais prévaut.
---
page_title: "pathly_runs"
description: |-
  Scenario executions, most recent first.
---

# pathly_runs

Lists executions. History, not desired state. Filter on a scenario whenever
possible: the unfiltered listing walks the whole organization.

```terraform
data "pathly_runs" "checkout_failures" {
  filter_scenario_id = pathly_scenario.checkout.id
  filter_status      = "fail"
}
```

`filter_scenario_id` is sent to the API and lowers the number of pages read.
`filter_status` is applied after the read (`ok`, `fail` or `error`).

## Schema

### Optional

- `filter_scenario_id` (String) — Passed to the API.
- `filter_status` (String) — `ok`, `fail` or `error`.

### Computed

- `runs` (List of Object) — See [`pathly_run`](run.md) for the attributes of
  one execution.

Required scope: `runs:read`.
