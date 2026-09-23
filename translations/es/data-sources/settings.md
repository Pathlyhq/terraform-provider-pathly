> **La versión inglesa es la de referencia.** Esta página traduce
> [`docs/data-sources/settings.md`](../../../docs/data-sources/settings.md), la única versión que lee el registro de Terraform. Si hay
> divergencia, prevalece la inglesa.
---
page_title: "pathly_settings"
description: |-
  Organization settings, read-only.
---

# pathly_settings (data source)

Reads the organization settings without taking ownership of them. Use the
`pathly_settings` resource to change them.

```terraform
data "pathly_settings" "this" {}

output "plan" {
  value = data.pathly_settings.this.plan_id
}
```

The attributes match the resource: alerting, status page, timezone, triage and
the seven integration flags. Required scope: `org:read`.
