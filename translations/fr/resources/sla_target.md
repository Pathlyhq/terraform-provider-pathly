---
page_title: "pathly_sla_target"
description: |-
  Objectif de disponibilité mesuré sur une fenêtre glissante, avec un budget d'erreur.
---

> **La version anglaise fait référence.** Cette page traduit
> [`docs/resources/sla_target.md`](../../../docs/resources/sla_target.md),
> seule version lue par le registre Terraform. En cas d'écart entre les deux,
> la version anglaise prévaut. Une traduction périmée qui annoncerait de
> mauvaises portées d'API conduirait à créer une clé trop privilégiée.

# pathly_sla_target

Un objectif de disponibilité, mesuré sur une fenêtre glissante avec un budget
d'erreur.

## Exemple

```terraform
resource "pathly_sla_target" "checkout" {
  scenario_id          = pathly_scenario.checkout.id
  name                 = "Checkout funnel — 99.9 %"
  objective_pct        = 99.9
  window_days          = 30
  exclude_maintenance  = true
  warn_at_budget_ratio = 0.8
}
```

Avec ces valeurs, l'objectif tolère environ 43 minutes d'indisponibilité par
période de 30 jours, et avertit quand 80 % de ce budget ont été consommés, soit
environ 35 minutes, alors qu'il reste encore de la marge pour agir.

## Schéma

### Obligatoire

- `objective_pct` (Number) — Disponibilité visée, de 50 à 100. Par exemple
  `99.9`.
- `window_days` (Number) — Fenêtre de mesure glissante, en jours, de 1 à 365.

### Optionnel

- `scenario_id` (String) — Scénario mesuré. Quand il est absent, l'objectif
  s'applique à toute l'organisation. **Force le remplacement** : un objectif
  est identifié par son scénario, le déplacer reviendrait à mesurer autre chose
  sous le même identifiant.
- `name` (String) — Libellé affiché dans les rapports. 120 caractères au plus.
- `exclude_maintenance` (Boolean) — Quand true, les fenêtres de maintenance ne
  consomment pas le budget d'erreur.
- `warn_at_budget_ratio` (Number) — Part du budget d'erreur consommée qui
  déclenche l'avertissement, de 0.1 à 1.
- `enabled` (Boolean) — Quand false, l'objectif reste défini mais n'avertit
  plus.

### Calculé

- `id` (String) — Identifiant attribué par Pathly.

## Création et modification

L'API expose un *upsert* : création et modification passent par le même appel,
identifié par le scénario visé. Un objectif déjà défini dans la console pour ce
scénario sera donc adopté par le premier `apply`, sans erreur de conflit.

## Import

```sh
terraform import pathly_sla_target.checkout sla_01H8ZK…
```
