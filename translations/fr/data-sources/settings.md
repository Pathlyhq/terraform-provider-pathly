> **La version anglaise fait référence.** Cette page traduit
> [`docs/data-sources/settings.md`](../../../docs/data-sources/settings.md), seule version lue par le registre Terraform. En cas
> d'écart, l'anglais prévaut.
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
