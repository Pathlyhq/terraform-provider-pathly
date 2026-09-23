---
page_title: "pathly_scenario"
description: |-
  An HTTP monitoring scenario.
---

# pathly_scenario

An HTTP monitoring scenario.

Browser journeys are not managed here: their steps carry login credentials,
which a Terraform file and its state would keep in plaintext. Create them in the
console.

## Example

```terraform
resource "pathly_scenario" "checkout" {
  name         = "Checkout funnel"
  url          = "https://shop.example.com/cart"
  interval_sec = 60

  expected_status = 200
  expect_text     = "Your cart"
  max_latency_ms  = 1500

  severity = "critical"
  folder   = "Shop"
  tags     = ["prod", "payment"]
  runbook  = "https://wiki.example.com/ops/checkout-unavailable"
}
```

## Schema

### Required

- `name` (String) — Name displayed in the console and in alerts. 1 to 120
  characters.
- `interval_sec` (Number) — Period between two runs, in seconds. `0` for a
  scenario driven by `cron` only. From 0 to 2,592,000.

### Optional

- `url` (String) — Monitored address, over http or https.
- `type` (String) — `http` only. **Forces replacement**: changing the type would
  destroy the scenario history.
- `enabled` (Boolean) — When false, the scenario exists but does not run.
- `method` (String) — `GET` or `HEAD`. A monitor that posts or deletes would act
  on the site at every run.
- `expected_status` (Number) — Expected HTTP status, from 100 to 599.
- `max_latency_ms` (Number) — Above this, the run is a performance failure.
  From 100 to 600,000.
- `expect_text` (String) — Text expected in the response. A 200 served by an
  error page is still a failure. 500 characters at most.
- `regions` (List of String) — Probe regions, 8 at most. When empty, Pathly
  picks the default region of the plan.
- `tags` (List of String) — Free-form tags, 10 at most, used to filter and
  group.
- `folder` (String) — Folder used to organize the console, 60 characters at most.
- `severity` (String) — `critical`, `major` or `minor`. Drives escalation.
- `runbook` (String) — On-call instructions, attached to the alert. 2,000
  characters at most.
- `cron` (String) — Cron schedule, in addition to or instead of `interval_sec`.

### Computed

- `id` (String) — Identifier assigned by Pathly.
- `last_status` (String) — Verdict of the last known run.
- `muted_until` (String) — Mute deadline, set from the console or the API. Read
  only here: a mute is a temporary operational gesture, not a desired state, and
  an `apply` must not wake up a scenario that was muted the night before.
- `created_at` (String) — Creation date.

## Attributes left empty

The optional attributes are also computed: when absent from the configuration,
they keep the value Pathly chose at creation time, or the one set from the
console. The plan therefore stays empty, there is no back and forth at every
`apply`.

The trade-off: **removing a line does not revert to the default**. Going from
`severity = "critical"` to nothing leaves the scenario on `critical`. To go back
to the default, set the value you want explicitly.

## Import

```sh
terraform import pathly_scenario.checkout mon_01H8ZK…
```

The identifier can be read from the scenario URL in the console. After the
import, `terraform plan` shows the differences between the configuration and
what the API returns: as long as it is not empty, the `apply` will modify the
existing scenario.
