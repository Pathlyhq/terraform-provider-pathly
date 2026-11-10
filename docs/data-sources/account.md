---
page_title: "pathly_account"
description: |-
  Organization and Space of the key, active plan, quotas used and remaining.
---

# pathly_account

Reads who the key belongs to and how much room the plan leaves: organization,
Space of the key, active plan, and each quota with what is used and what
remains. Useful as a `precondition` before adding Flow or Spaces the plan does
not cover.

## Example

```terraform
data "pathly_account" "this" {}

check "flow_room" {
  assert {
    # A null remaining means the plan sets no quota.
    condition     = coalesce(data.pathly_account.this.quotas.flow.remaining, 1) > 0
    error_message = "No Flow left on the plan. See ${data.pathly_account.this.billing_url}."
  }
}
```

## Schema

### Computed

- `organization_id` (String) — Identifier of the organization. It is also the
  identifier of its status page, to import into `pathly_status_page`.
- `organization_name` (String) — Name of the organization.
- `space_id` (String) — Space the key is limited to. Null for a key of the
  whole organization.
- `space_name` (String) — Name of that Space, null otherwise.
- `plan_id` (String) — Active plan.
- `plan_name` (String) — Display name of the plan.
- `plan_price_monthly_eur` (Number) — Monthly list price of the plan, in euros
  excluding VAT.
- `billing_url` (String) — Plan screen of the app, to upgrade or buy runs.
- `quotas` (Object) — Quotas of the plan:
  - `ping`, `chain`, `flow` — active Ping (scheduled heartbeats included),
    Chain and Flow, each with `used`, `limit` and `remaining`.
  - `runs` — browser runs of the month: `used`, `limit`, `remaining`,
    `period` (`month`) and `pack_runs_remaining`, the runs still available in
    the packs on top of the plan.
  - `assistant_trials` — Flow assistant trials of the day: `used`, `limit`,
    `remaining` and `period` (`day`).
  - `spaces` — active Spaces, with `used`, `limit` and `remaining`. Null for a
    key limited to a Space, and while Spaces are not open for the
    organization.

A null `limit` (and `remaining`) means the plan sets no quota. Every counter is
a number, as in `pathly_usage`.

For a key limited to a Space, `used` only counts that Space, while `limit` and
`remaining` stay those of the organization, whose quotas the Space shares.

## Required scope

`org:read`.
