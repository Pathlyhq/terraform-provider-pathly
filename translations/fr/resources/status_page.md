---
page_title: "pathly_status_page"
description: |-
  La page de statut de l'organisation, ou celle d'un Espace, avec son badge.
---

> **La version anglaise fait référence.** Cette page traduit
> [`docs/resources/status_page.md`](../../../docs/resources/status_page.md),
> seule version lue par le registre Terraform. En cas d'écart entre les deux,
> la version anglaise prévaut.

# pathly_status_page

La page de statut de l'organisation (sans `space_id`) ou celle d'un Espace
(avec `space_id`), avec l'adresse de son badge SVG.

La création reprend une page qui existe déjà : l'API la rend telle quelle et
applique `public`. Un premier apply reprend donc la page créée dans la console,
sans import.

## Exemple

```terraform
resource "pathly_status_page" "shop" {
  space_id = pathly_space.shop.id
  public   = true
}

output "shop_badge" {
  value = pathly_status_page.shop.badge_url
}
```

## Schéma

- `space_id` (String, optionnel) — Espace dont c'est la page. Absent pour la
  page de l'organisation. **Force le remplacement.**
- `slug` (String, optionnel) — Adresse de la page de l'organisation, proposée
  d'après son nom si absente. **La changer casse les liens et badges déjà
  posés.** Refusé pour une page d'Espace, qui prend l'adresse de son Espace.
- `public` (Boolean, optionnel) — Publie la page. Une page neuve reste privée
  sauf `true`. Publier expose les noms des scénarios et leur disponibilité.
- Calculés : `id` (celui de l'organisation ou de l'Espace), `kind`
  (`organization` ou `space`), `name`, `url` et `badge_url` (ajoutez
  `?v=uptime` pour la disponibilité). `url` et `badge_url` gardent leur valeur
  dans le plan tant que `slug` ne change pas.

## Détruire une page de statut la dépublie

L'API n'a pas de suppression pour une page de statut. Un destroy **dépublie**
donc la page (`public = false`), garde son adresse et émet un avertissement.
Terraform cesse de la suivre. La republier plus tard fait revenir les badges
déjà collés sur les sites des clients.

## Page de l'organisation et `pathly_settings`

`status_slug` et `status_public` de `pathly_settings` pilotent la même page de
l'organisation. Gérez-la à un seul endroit. Les pages d'Espace ne se gèrent
qu'ici.

## Import

L'identifiant de la page de l'organisation est celui de l'organisation
(`data.pathly_account.this.organization_id`). Celui d'une page d'Espace est
celui de l'Espace.

```sh
terraform import pathly_status_page.shop 6f1c2a9e-3b7d-4c58-a0e4-9d2f5b8c1a67
```

## Portées requises

`status:read` pour rafraîchir, `status:write` pour créer, publier, dépublier et
déplacer.
