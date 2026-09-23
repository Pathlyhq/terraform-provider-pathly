---
page_title: "Pathly Provider"
description: |-
  Gérez la surveillance Pathly comme du code : scénarios, fenêtres de maintenance, webhooks et objectifs de disponibilité.
---

> **La version anglaise fait référence.** Cette page traduit
> [`docs/index.md`](../../docs/index.md), seule version lue par le registre
> Terraform. En cas d'écart entre les deux, la version anglaise prévaut. Une
> traduction périmée qui annoncerait de mauvaises portées d'API conduirait à
> créer une clé trop privilégiée.

# Pathly Provider

Décrivez la surveillance Pathly comme le reste de votre infrastructure : relue
dans une *pull request*, appliquée par la CI, identique d'un environnement à
l'autre.

```terraform
terraform {
  required_providers {
    pathly = {
      source  = "pathlyhq/pathly"
      version = "~> 0.1"
    }
  }
}

provider "pathly" {}

resource "pathly_scenario" "checkout" {
  name         = "Checkout"
  url          = "https://shop.example.com/cart"
  interval_sec = 300
  expect_text  = "Your cart"
  severity     = "critical"
}
```

## Authentification

```sh
export PATHLY_API_TOKEN="sp_…"
```

La clé se crée sous **Settings → API keys** (Paramètres → Clés d'API), depuis
un compte propriétaire ou administrateur. Donnez-lui les portées strictement
nécessaires :

| Ressources gérées | Portées |
|---|---|
| `pathly_scenario`, `pathly_scenarios` | `scenarios:read`, `scenarios:write` |
| `pathly_settings` | `org:read`, `org:write` |
| `pathly_webhook` | `alerting:read`, `alerting:write` |
| `pathly_maintenance_window` | `maintenance:read`, `maintenance:write` |
| `pathly_sla_target` | `sla:read`, `sla:write` |
| `pathly_incidents` | `incidents:read` |
| `pathly_members` | `members:read` |
| `pathly_runs`, `pathly_run` | `runs:read` |
| `pathly_sla`, `pathly_sla_targets` | `sla:read` |
| `pathly_usage` | `org:read` |

Pour un pipeline qui ne fait que `terraform plan`, les portées `:read`
suffisent. Aucune portée d'organisation n'est exigée : la vérification faite au
moment de la configuration tolère son refus.

## Schéma

### Optionnel

- `api_token` (String, sensible) — Clé d'API de l'organisation, préfixe `sp_`.
  Mieux vaut la fournir par `PATHLY_API_TOKEN` : écrite dans un fichier `.tf`
  elle finit dans le dépôt, et lue depuis une variable Terraform elle finit en
  clair dans le fichier d'état.
- `api_url` (String) — Base de l'API. Vaut `https://api.pathlyhq.com` par
  défaut. Le HTTP en clair n'est accepté que sur `localhost` : partout
  ailleurs, la clé circulerait lisible dans l'en-tête `Authorization`.

Les variables d'environnement `PATHLY_API_TOKEN` et `PATHLY_API_URL` priment
sur les attributs du bloc.

## Comportement

- **Configuration** : un appel de vérification est fait une fois. Une clé
  expirée, révoquée ou tronquée échoue ici, avec un message qui dit quoi
  corriger, au lieu d'une erreur par ressource pendant le plan. Une clé valide
  mais aux portées limitées passe : un refus de cet appel n'est pas une erreur.
- **Limite de débit** : l'en-tête `Retry-After` est honoré, avec un plafond de
  90 secondes et quatre tentatives.
- **Créations** : chaque création porte une clé d'idempotence. Si la réponse
  est perdue, la nouvelle tentative renvoie la ressource déjà créée plutôt
  qu'une seconde copie.
- **Ressource disparue** : elle quitte l'état et le plan suivant la recrée. Un
  refus ou une panne n'est jamais pris pour une disparition.

## Hors périmètre, délibérément

- **Clés d'API** — il n'existe pas d'endpoint de jeton sur `/v1`. Une clé qui
  peut en forger d'autres transformerait une fuite en prise de contrôle.
- **Les membres se lisent, ils ne s'écrivent pas** — inviter un propriétaire
  depuis un fichier Terraform transformerait une écriture dans le dépôt en
  octroi d'accès.
- **Trois actions** — déclencher une exécution, réinitialiser une baseline de
  comparaison, et résoudre un incident. Terraform rejoue un état désiré : un
  `apply` une semaine plus tard les relancerait. Elles restent dans l'API et
  la console.
- **Mise en sourdine** — geste opérationnel temporaire. `muted_until` est en
  lecture seule.
