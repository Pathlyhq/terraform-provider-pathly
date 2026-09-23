> **La version anglaise fait référence.** Cette page traduit
> [`docs/data-sources/run.md`](../../../docs/data-sources/run.md), seule version lue par le registre Terraform. En cas
> d'écart, l'anglais prévaut.
---
page_title: "pathly_run"
description: |-
  A single scenario execution and its verdict.
---

# pathly_run

Reads one execution by identifier. The API answers a narrower projection here
than in the listing, so a few attributes come back empty.

```terraform
data "pathly_run" "last_failure" {
  id = data.pathly_runs.checkout_failures.runs[0].id
}
```

The step-by-step results and the triage signals are left out: they are
free-form JSON whose shape follows the scenario, and nobody can plan against
them. They stay in the console.

## Schema

### Required

- `id` (String) — Identifier of the execution.

### Computed

- `scenario_id`, `scenario_name`, `scenario_type` (String)
- `status` (String) — `ok`, `fail` or `error`.
- `latency_ms` (Number)
- `message` (String) — Reason for the failure, when there is one.
- `probe_region` (String)
- `compared_to_run_id`, `confirms_run_id` (String)
- `triage_verdict` (String) — `real`, `suspect` or `benign`.
- `triage_reason` (String)
- `in_maintenance` (Boolean) — The run happened inside a window and does not
  count against the SLA.
- `browser_seconds` (Number)
- `screenshot_path`, `video_path` (String)
- `created_at` (String)

Required scope: `runs:read`.
