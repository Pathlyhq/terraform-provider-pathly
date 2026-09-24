---
page_title: "pathly_maintenance_window"
description: |-
  Fenêtre de maintenance, ponctuelle ou hebdomadaire. Les échecs qui s'y produisent ne déclenchent aucune alerte.
---

> **La version anglaise fait référence.** Cette page traduit
> [`docs/resources/maintenance_window.md`](../../../docs/resources/maintenance_window.md),
> seule version lue par le registre Terraform. En cas d'écart entre les deux,
> la version anglaise prévaut. Une traduction périmée qui annoncerait de
> mauvaises portées d'API conduirait à créer une clé trop privilégiée.

# pathly_maintenance_window

Une fenêtre pendant laquelle les échecs n'alertent pas et, si les objectifs de
disponibilité le demandent, ne consomment pas le budget d'erreur.

Deux formes, mutuellement exclusives :

- **weekly** — `weekday`, `start_minute` et `duration_min`,
- **one-off** — `starts_at` et `ends_at`.

Sans `scenario_id`, la fenêtre couvre toute l'organisation.

~> **Pas de modification sur place.** L'API ne sait pas réécrire une fenêtre :
tout changement détruit la ressource et la recrée. C'est visible dans le plan.

## Exemples

```terraform
# Backup, every Sunday from 3 am to 5 am.
resource "pathly_maintenance_window" "backup" {
  weekday      = 7
  start_minute = 180
  duration_min = 120
  reason       = "Weekly backup"
}

# Announced migration, on a single scenario.
resource "pathly_maintenance_window" "migration" {
  scenario_id = pathly_scenario.checkout.id
  starts_at   = "2026-10-04T22:00:00Z"
  ends_at     = "2026-10-05T02:00:00Z"
  reason      = "Checkout funnel migration"
}
```

## Schéma

### Optionnel

Chaque attribut **force le remplacement**.

- `scenario_id` (String) — Scénario visé. Quand il est absent, la fenêtre
  couvre toute l'organisation.
- `starts_at` (String) — Début, en ISO 8601. Forme ponctuelle.
- `ends_at` (String) — Fin, en ISO 8601. Forme ponctuelle. L'API refuse une
  fenêtre ponctuelle de plus de 90 jours : une fenêtre qui s'éternise masque de
  vraies pannes.
- `weekday` (Number) — Jour de la semaine, 1 pour lundi jusqu'à 7 pour
  dimanche. Forme hebdomadaire.
- `start_minute` (Number) — Minute de début dans la journée, de 0 à 1439 (heure
  UTC). `180` correspond à 3 h du matin.
- `duration_min` (Number) — Durée en minutes, de 15 à 1440.
- `reason` (String) — Motif affiché dans la console et dans la piste d'audit.
  500 caractères au plus.

### Calculé

- `id` (String) — Identifiant attribué par Pathly.

## Import

Préférez un bloc `import` (Terraform / OpenTofu ≥ 1.5).

```terraform
import {
  to = pathly_maintenance_window.backup
  id = "mw_01H8ZK…"
}
```

```sh
terraform import pathly_maintenance_window.backup mw_01H8ZK…
```
