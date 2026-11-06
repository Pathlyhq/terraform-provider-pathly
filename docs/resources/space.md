---
page_title: "pathly_space"
description: |-
  A Space: the scenarios of one client, one brand or one application.
---

# pathly_space

A Space groups the scenarios of one client, one brand or one application. An
agency describes one Space per client, files the client sites in it with
`space_id` on `pathly_scenario`, and publishes a status page per client with
`pathly_status_page`.

Spaces must be open for the organization. Until then the API answers 404 on
every Space route: a creation fails with that error, and a Space already in
the state leaves it on the next refresh.

## Example

```terraform
resource "pathly_space" "shop" {
  name = "Example shop"
  kind = "client"
}

resource "pathly_scenario" "shop_home" {
  name         = "Example shop — home page"
  url          = "https://shop.example.com"
  interval_sec = 300
  space_id     = pathly_space.shop.id
}
```

Full working example: [`examples/agency`](../../examples/agency/).

## Schema

### Required

- `name` (String) — Name of the Space, 1 to 80 characters, unique among the
  active Spaces of the organization. A rename also renames the status page
  component of the scenarios filed in it.

### Optional

- `kind` (String) — `client`, `site` or `application`. When absent, Pathly
  derives it from the usage profile of the organization.

### Computed

- `id` (String) — Identifier assigned by Pathly.
- `slug` (String) — Address of the Space, derived from its name at creation and
  kept on rename. Its status page lives under the address of the organization
  page.
- `status_public` (Boolean) — Whether the status page of the Space is
  published. Managed by `pathly_status_page`, read only here.
- `created_at` (String) — Creation date.

## Destroying a Space archives it

The API has no deletion for a Space: `DELETE /v1/spaces/{id}` archives it. Its
scenarios leave it and **keep running**, nothing is deleted, and the name
becomes free again. To stop monitoring a client, destroy its `pathly_scenario`
resources as well.

The plan limits the number of active Spaces. An archived Space does not count.
Read the limit from `data.pathly_account.this.quotas.spaces` or from
`data.pathly_spaces.all.max_spaces`.

## Import

```terraform
import {
  to = pathly_space.shop
  id = "6f1c2a9e-…"
}
```

```sh
terraform import pathly_space.shop 6f1c2a9e-…
```

The identifiers are listed by the `pathly_spaces` data source.

## Required scopes

`org:read` to refresh, `org:write` to create, rename and archive. A key limited
to a Space cannot create one.
