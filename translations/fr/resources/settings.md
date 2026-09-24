---
page_title: "pathly_settings"
description: |-
  Paramètres d'organisation : alertes, page de statut, fuseau et triage.
---

> **La version anglaise fait référence.** Cette page traduit
> [`docs/resources/settings.md`](../../../docs/resources/settings.md), seule
> version lue par le registre Terraform.

# pathly_settings

Paramètres d'organisation : alertes, escalade, page de statut, fuseau et
triage.

Il n'existe qu'un objet par organisation. Le déclarer deux fois ferait
s'écraser les deux définitions à chaque apply. Create et Update sont le même
appel : les paramètres existent avant Terraform.

`terraform destroy` **ne les remet pas à zéro**. Il n'y a pas d'endpoint de
suppression, et il ne doit pas y en avoir : détruire un workspace de test
couperait silencieusement les alertes de toute l'organisation.

## Exemple simple

```terraform
resource "pathly_settings" "this" {
  timezone    = "Europe/Paris"
  alert_email = "ops@example.com"
}
```

## Exemple complexe

```terraform
resource "pathly_settings" "this" {
  timezone               = "Europe/Paris"
  alert_email            = "ops@example.com"
  alert_on_recovery      = true
  escalation_after_fails = 3
  escalation_email       = "oncall@example.com"
  weekly_digest_enabled  = true
  weekly_digest_email    = "ops@example.com"

  status_slug   = "acme"
  status_public = true

  ssl_warn_days    = [30, 14, 7]
  domain_warn_days = [30]

  triage_enabled         = true
  triage_confirm_enabled = true
  triage_latency_factor  = 2.5
}
```

Activer `status_public` publie les noms des scénarios et leur disponibilité.
Vérifiez qu'aucun ne nomme un hôte interne.

Le schéma détaillé est dans la [version anglaise](../../../docs/resources/settings.md).
Portées : `org:read` pour rafraîchir, `org:write` pour modifier.

```terraform
import {
  to = pathly_settings.this
  id = "settings"
}
```

```sh
terraform import pathly_settings.this settings
```
