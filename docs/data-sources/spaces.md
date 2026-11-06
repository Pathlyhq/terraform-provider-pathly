---
page_title: "pathly_spaces"
description: |-
  Active Spaces of the organization, with the limit of the plan.
---

# pathly_spaces

Lists the active Spaces of the organization, sorted by name, with the number
of Spaces the plan allows. A key limited to a Space only sees its own.

Main use: adopting the Spaces an agency already created in the console, and
filing scenarios into them by name.

## Example

```terraform
data "pathly_spaces" "all" {}

locals {
  space_by_name = { for s in data.pathly_spaces.all.spaces : s.name => s.id }
}

resource "pathly_scenario" "shop_home" {
  name         = "Example shop — home page"
  url          = "https://shop.example.com"
  interval_sec = 300
  space_id     = local.space_by_name["Example shop"]
}
```

## Schema

### Computed

- `max_spaces` (Number) — Active Spaces the plan allows. Null when the plan
  sets no limit.
- `spaces` (List of Object) — Active Spaces, each with `id`, `name`, `slug`,
  `kind`, `status_public`, `scenario_count` and `created_at`. Archived Spaces
  are not listed.

## Spaces not open

While Spaces are not open for the organization, the API answers 404 and the
data source fails with an explicit error. An empty list would read as "no
client", and feed a `for_each` that destroys.

## Required scope

`org:read`.
