---
page_title: "pathly_space"
description: |-
  Un Espace : les scénarios d'un client, d'une marque ou d'une application.
---

> **La version anglaise fait référence.** Cette page traduit
> [`docs/resources/space.md`](../../../docs/resources/space.md), seule version
> lue par le registre Terraform. En cas d'écart entre les deux, la version
> anglaise prévaut.

# pathly_space

Un Espace regroupe les scénarios d'un client, d'une marque ou d'une
application. Une agence décrit un Espace par client, y range les sites du
client avec `space_id` sur `pathly_scenario`, et publie une page de statut par
client avec `pathly_status_page`.

Les Espaces doivent être ouverts pour l'organisation. Avant cela, l'API répond
404 sur toutes les routes des Espaces : une création échoue avec cette erreur,
et un Espace déjà dans l'état en sort au rafraîchissement suivant.

## Exemple

```terraform
resource "pathly_space" "shop" {
  name = "Example shop"
  kind = "client"
}

resource "pathly_scenario" "shop_home" {
  name         = "Example shop — page d'accueil"
  url          = "https://shop.example.com"
  interval_sec = 300
  space_id     = pathly_space.shop.id
}
```

Exemple complet : [`examples/agency`](../../../examples/agency/).

## Schéma

- `name` (String, obligatoire) — Nom de l'Espace, 1 à 80 caractères, unique
  parmi les Espaces actifs de l'organisation.
- `kind` (String, optionnel) — `client`, `site` ou `application`. Absent, Pathly
  le déduit du profil d'usage de l'organisation.
- Calculés : `id`, `slug` (adresse de l'Espace, gardée au renommage),
  `status_public` (page de l'Espace publiée ou non, gérée par
  `pathly_status_page`) et `created_at`.

## Détruire un Espace l'archive

L'API n'a pas de suppression pour un Espace : `DELETE /v1/spaces/{id}`
l'archive. Ses scénarios en sortent et **restent surveillés**, rien n'est
supprimé, et le nom redevient libre. Pour arrêter la surveillance d'un client,
détruisez aussi ses ressources `pathly_scenario`.

## Import

```sh
terraform import pathly_space.shop 6f1c2a9e-…
```

Les identifiants sont listés par la data source `pathly_spaces`.

## Portées requises

`org:read` pour rafraîchir, `org:write` pour créer, renommer et archiver. Une
clé limitée à un Espace ne peut pas en créer.
