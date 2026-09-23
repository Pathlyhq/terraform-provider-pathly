> **La version anglaise fait référence.** Cette page traduit
> [`docs/data-sources/usage.md`](../../../docs/data-sources/usage.md), seule version lue par le registre Terraform. En cas
> d'écart, l'anglais prévaut.
---
page_title: "pathly_usage"
description: |-
  Current plan consumption.
---

# pathly_usage

Reads the counters of the current billing period. Useful as a `precondition`
before adding browser scenarios that would exhaust the pack.

```terraform
data "pathly_usage" "this" {}

check "browser_budget" {
  assert {
    condition     = data.pathly_usage.this.pack_runs_remaining > 100
    error_message = "Fewer than 100 browser runs left this period."
  }
}
```

## Schema

### Computed

- `plan_id` (String) — Active plan.
- `browser_runs_used` (Number) — Browser runs already consumed.
- `pack_runs_used_this_period` (Number) — Pack runs consumed this period.
- `pack_runs_remaining` (Number) — Pack runs still available.

Required scope: `org:read`.
