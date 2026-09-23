> **La versión inglesa es la de referencia.** Esta página traduce
> [`docs/data-sources/status_page.md`](../../../docs/data-sources/status_page.md), la única versión que lee el registro de Terraform. Si hay
> divergencia, prevalece la inglesa.
---
page_title: "pathly_status_page"
description: |-
  Public status page and its components.
---

# pathly_status_page

Reads the public status page. A missing or private page is an error, not an
empty page: confusing the two would hide a `status_public` that was never
turned on.

```terraform
data "pathly_status_page" "public" {}

output "components" {
  value = [
    for c in data.pathly_status_page.public.components : {
      name       = c.name
      uptime_30d = c.uptime_30d
    }
  ]
}
```

Set `status_slug` and `status_public` on `pathly_settings` before reading this
data source.

## Schema

### Computed

- `uptime_30d` (Number) — Overall availability over 30 days.
- `components` (List of Object) — `name`, `uptime_30d`, `uptime_90d`. A
  component without 90 days of history leaves `uptime_90d` null, not zero.

Required scope: none beyond a valid key. The page is public by design.
