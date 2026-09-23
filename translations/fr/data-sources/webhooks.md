> **La version anglaise fait référence.** Cette page traduit
> [`docs/data-sources/webhooks.md`](../../../docs/data-sources/webhooks.md), seule version lue par le registre Terraform. En cas
> d'écart, l'anglais prévaut.
---
page_title: "pathly_webhooks"
description: |-
  Outbound webhooks, without their destination.
---

# pathly_webhooks

Lists the webhooks. The destination and the signing secret are never returned:
the state of a listing must not recover a URL that a leaked backend would
expose.

```terraform
data "pathly_webhooks" "all" {}

output "destinations" {
  value = [for h in data.pathly_webhooks.all.webhooks : h.url_fingerprint]
}
```

## Schema

### Computed

- `webhooks` (List of Object) — `id`, `events`, `enabled`, `has_secret`,
  `url_fingerprint`, `created_at`.

Required scope: `alerting:read`.
