---
page_title: "pathly_scenario"
description: |-
  Un scénario de surveillance HTTP.
---

> **La version anglaise fait référence.** Cette page traduit
> [`docs/resources/scenario.md`](../../../docs/resources/scenario.md), seule
> version lue par le registre Terraform. En cas d'écart entre les deux, la
> version anglaise prévaut. Une traduction périmée qui annoncerait de mauvaises
> portées d'API conduirait à créer une clé trop privilégiée.

# pathly_scenario

Un scénario de surveillance HTTP.

Les parcours navigateur ne se gèrent pas ici : leurs étapes portent des
identifiants de connexion, qu'un fichier Terraform et son état conserveraient
en clair. Créez-les dans la console.

## Exemple

```terraform
resource "pathly_scenario" "checkout" {
  name         = "Checkout funnel"
  url          = "https://shop.example.com/cart"
  interval_sec = 60

  expected_status = 200
  expect_text     = "Your cart"
  max_latency_ms  = 1500

  severity = "critical"
  folder   = "Shop"
  tags     = ["prod", "payment"]
  runbook  = "https://wiki.example.com/ops/checkout-unavailable"
}
```

## Schéma

### Obligatoire

- `name` (String) — Nom affiché dans la console et dans les alertes. De 1 à 120
  caractères.
- `interval_sec` (Number) — Période entre deux exécutions, en secondes. `0`
  pour un scénario piloté uniquement par `cron`. De 0 à 2 592 000.

### Optionnel

- `url` (String) — Adresse surveillée, en http ou en https.
- `type` (String) — `http` uniquement. **Force le remplacement** : changer le
  type détruirait l'historique du scénario.
- `enabled` (Boolean) — Quand false, le scénario existe mais ne s'exécute pas.
- `method` (String) — `GET` ou `HEAD`. Un moniteur qui poste ou supprime
  agirait sur le site à chaque exécution.
- `expected_status` (Number) — Statut HTTP attendu, de 100 à 599.
- `max_latency_ms` (Number) — Au-delà, l'exécution est un échec de performance.
  De 100 à 600 000.
- `expect_text` (String) — Texte attendu dans la réponse. Un 200 servi par une
  page d'erreur reste un échec. 500 caractères au plus.
- `regions` (List of String) — Régions de sonde, 8 au plus. Quand la liste est
  vide, Pathly choisit la région par défaut du plan.
- `tags` (List of String) — Étiquettes libres, 10 au plus, servant à filtrer et
  à regrouper.
- `folder` (String) — Dossier servant à organiser la console, 60 caractères au
  plus.
- `severity` (String) — `critical`, `major` ou `minor`. Pilote l'escalade.
- `runbook` (String) — Consignes d'astreinte, jointes à l'alerte. 2 000
  caractères au plus.
- `cron` (String) — Planning cron, en complément ou à la place de
  `interval_sec`.

### Calculé

- `id` (String) — Identifiant attribué par Pathly.
- `last_status` (String) — Verdict de la dernière exécution connue.
- `muted_until` (String) — Échéance de la mise en sourdine, posée depuis la
  console ou l'API. En lecture seule ici : une mise en sourdine est un geste
  opérationnel temporaire, pas un état désiré, et un `apply` ne doit pas
  réveiller un scénario mis en sourdine la veille au soir.
- `created_at` (String) — Date de création.

## Attributs laissés vides

Les attributs optionnels sont aussi calculés : absents de la configuration, ils
conservent la valeur que Pathly a choisie à la création, ou celle posée depuis
la console. Le plan reste donc vide, il n'y a pas d'aller-retour à chaque
`apply`.

La contrepartie : **retirer une ligne ne revient pas à la valeur par défaut**.
Passer de `severity = "critical"` à rien laisse le scénario sur `critical`.
Pour revenir à la valeur par défaut, posez explicitement la valeur voulue.

## Import

```sh
terraform import pathly_scenario.checkout mon_01H8ZK…
```

L'identifiant se lit dans l'URL du scénario dans la console. Après l'import,
`terraform plan` montre les différences entre la configuration et ce que
renvoie l'API : tant qu'il n'est pas vide, l'`apply` modifiera le scénario
existant.
