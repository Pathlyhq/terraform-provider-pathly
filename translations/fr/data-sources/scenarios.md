---
page_title: "pathly_scenarios"
description: |-
  Inventaire des scénarios de l'organisation, filtrable par étiquette ou par dossier.
---

> **La version anglaise fait référence.** Cette page traduit
> [`docs/data-sources/scenarios.md`](../../../docs/data-sources/scenarios.md),
> seule version lue par le registre Terraform. En cas d'écart entre les deux,
> la version anglaise prévaut. Une traduction périmée qui annoncerait de
> mauvaises portées d'API conduirait à créer une clé trop privilégiée.

# pathly_scenarios

Liste les scénarios de l'organisation. Toutes les pages sont parcourues : une
source de données qui s'arrêterait à la première produirait des boucles
`for_each` silencieusement incomplètes.

Deux usages courants : adopter un parc existant sans tout réécrire, et repérer
les scénarios créés à la main qui échappent aux relectures.

## Exemple

```terraform
data "pathly_scenarios" "prod" {
  filter_tag = "prod"
}

# What the console holds and the code ignores.
output "scenarios_outside_terraform" {
  value = [
    for s in data.pathly_scenarios.prod.scenarios : s.name
    if !contains([for j in pathly_scenario.journey : j.id], s.id)
  ]
}
```

## Schéma

### Optionnel

- `filter_tag` (String) — Ne garde que les scénarios portant cette étiquette.
- `filter_folder` (String) — Ne garde que les scénarios de ce dossier.

Les deux filtres se combinent, et s'appliquent après la lecture : ils ne
réduisent pas le nombre d'appels à l'API.

### Calculé

- `scenarios` (List of Object) — Scénarios retenus, chacun avec `id`, `name`,
  `type`, `url`, `enabled`, `interval_sec`, `folder`, `severity`, `tags` et
  `last_status`.

## Portée requise

`scenarios:read`. Un refus remonte comme une erreur : renvoyer une liste vide
détruirait tout un parc au prochain `apply`, si la sortie alimente un
`for_each`.
