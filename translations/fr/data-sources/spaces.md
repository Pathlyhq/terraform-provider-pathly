---
page_title: "pathly_spaces"
description: |-
  Espaces actifs de l'organisation, avec la limite de l'offre.
---

> **La version anglaise fait référence.** Cette page traduit
> [`docs/data-sources/spaces.md`](../../../docs/data-sources/spaces.md), seule
> version lue par le registre Terraform. En cas d'écart entre les deux, la
> version anglaise prévaut.

# pathly_spaces

Liste les Espaces actifs de l'organisation, triés par nom, avec le nombre
d'Espaces que permet l'offre. Une clé limitée à un Espace ne voit que le sien.

```terraform
data "pathly_spaces" "all" {}

locals {
  space_by_name = { for s in data.pathly_spaces.all.spaces : s.name => s.id }
}
```

## Schéma

- `max_spaces` (Number) — Espaces actifs permis par l'offre. Nul si l'offre ne
  fixe pas de limite.
- `spaces` (List of Object) — Espaces actifs, chacun avec `id`, `name`, `slug`,
  `kind`, `status_public`, `scenario_count` et `created_at`. Les Espaces
  archivés n'y sont pas.

Tant que les Espaces ne sont pas ouverts pour l'organisation, l'API répond 404
et la data source échoue avec une erreur explicite : une liste vide se lirait
comme « aucun client » et nourrirait un `for_each` qui détruit.

Portée requise : `org:read`.
