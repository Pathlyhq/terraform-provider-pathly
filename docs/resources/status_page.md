---
page_title: "pathly_status_page"
description: |-
  The status page of the organization, or that of a Space, with its badge.
---

# pathly_status_page

The status page of the organization (without `space_id`) or that of a Space
(with `space_id`), with the address of its SVG badge.

The creation adopts a page that already exists: the API returns it as it is,
and applies `public`. A first apply therefore takes over the page the console
created, without an import.

## Example

```terraform
resource "pathly_status_page" "shop" {
  space_id = pathly_space.shop.id
  public   = true
}

output "shop_badge" {
  value = pathly_status_page.shop.badge_url
}
```

Full working example: [`examples/agency`](../../examples/agency/).

## Schema

### Optional

- `space_id` (String) — Space whose page this is. Absent for the page of the
  organization. **Forces replacement.** A key limited to a Space can only
  manage the page of its Space.
- `slug` (String) — Address of the organization page: lowercase letters,
  digits and single dashes, 64 characters at most. Proposed from the name of
  the organization when absent. **Changing it breaks the links and badges
  already posted.** A Space page takes the address of its Space and refuses
  this attribute. An address taken by another organization is refused.
- `public` (Boolean) — Publishes the page. A new page stays private unless this
  is true. Publishing exposes the scenario names and their availability: check
  that none names an internal host.

### Computed

- `id` (String) — Identifier of the page: that of the organization for its
  page, that of the Space for a Space page.
- `kind` (String) — `organization` or `space`.
- `name` (String) — Name shown on the page: that of the organization or of the
  Space.
- `url` (String) — Public address of the page.
- `badge_url` (String) — Address of the SVG badge. Add `?v=uptime` for the
  availability. On the organization page, add `?monitor=<scenario id>` when it
  holds several scenarios.

`url` and `badge_url` keep their value in the plan unless `slug` changes.

## Destroying a status page unpublishes it

The API has no deletion for a status page: the page of the organization and
those of the Spaces exist as long as their owner does. A destroy therefore
**unpublishes** the page (`public = false`), keeps its address, and emits a
warning. Terraform stops tracking it. Publishing it again later brings back the
badges already pasted on client sites.

## The organization page and `pathly_settings`

`status_slug` and `status_public` on `pathly_settings` drive the same
organization page. Manage it in one place only: with both, each apply undoes
what the other wrote. Space pages are only managed here.

## Import

The identifier of the organization page is that of the organization, read from
`data.pathly_account.this.organization_id`. The identifier of a Space page is
that of the Space.

```terraform
import {
  to = pathly_status_page.org
  id = "0b7e…" # organization id
}
```

```sh
terraform import pathly_status_page.shop 6f1c2a9e-…   # Space id
```

## Required scopes

`status:read` to refresh, `status:write` to create, publish, unpublish and
move.
