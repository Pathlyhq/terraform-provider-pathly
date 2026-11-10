---
page_title: "pathly_account"
description: |-
  Organisation et Espace de la clé, offre active, quotas utilisés et restants.
---

> **La version anglaise fait référence.** Cette page traduit
> [`docs/data-sources/account.md`](../../../docs/data-sources/account.md),
> seule version lue par le registre Terraform. En cas d'écart entre les deux,
> la version anglaise prévaut.

# pathly_account

Lit à qui appartient la clé et la marge que laisse l'offre : organisation,
Espace de la clé, offre active, et chaque quota avec l'utilisé et le restant.
Utile en `precondition` avant d'ajouter des Flow ou des Espaces que l'offre ne
couvre pas.

```terraform
data "pathly_account" "this" {}

check "flow_room" {
  assert {
    # Un restant nul veut dire que l'offre ne fixe pas de quota.
    condition     = coalesce(data.pathly_account.this.quotas.flow.remaining, 1) > 0
    error_message = "Plus de Flow disponible sur l'offre. Voir ${data.pathly_account.this.billing_url}."
  }
}
```

## Schéma

- `organization_id`, `organization_name` — Organisation. Son identifiant est
  aussi celui de sa page de statut, à importer dans `pathly_status_page`.
- `space_id`, `space_name` — Espace auquel la clé est limitée, nuls pour une
  clé de toute l'organisation.
- `plan_id`, `plan_name`, `plan_price_monthly_eur` — Offre active et son prix
  mensuel catalogue, en euros HT.
- `billing_url` — Écran Offre de l'app.
- `quotas` — `ping`, `chain`, `flow`, `runs` (avec `period` et
  `pack_runs_remaining`), `assistant_trials` (avec `period`) et `spaces` (nul
  pour une clé d'Espace, et tant que les Espaces ne sont pas ouverts), chacun
  avec `used`, `limit` et `remaining`. Un `limit` nul veut dire sans quota.

Pour une clé limitée à un Espace, `used` ne compte que cet Espace, tandis que
`limit` et `remaining` restent ceux de l'organisation.

Portée requise : `org:read`.
