---
page_title: "pathly_scenario"
description: |-
  Un contrôle HTTP ou un parcours navigateur.
---

> **La version anglaise fait référence.** Cette page traduit
> [`docs/resources/scenario.md`](../../../docs/resources/scenario.md), seule
> version lue par le registre Terraform. En cas d'écart entre les deux, la
> version anglaise prévaut. Une traduction périmée qui annoncerait de mauvaises
> portées d'API conduirait à créer une clé trop privilégiée.

# pathly_scenario

Un contrôle HTTP ou un parcours navigateur.

Un parcours navigateur est en écriture seule : l'API ne renvoie jamais les
étapes, seulement `scenario_fingerprint`. Une étape `fill` ou `http_auth` peut
porter un mot de passe — gardez ces valeurs dans un coffre, pas dans le dépôt.
L'état les conserve tout de même, marquées sensibles.

## Exemple simple — contrôle HTTP

```terraform
resource "pathly_scenario" "home" {
  name         = "Home page"
  url          = "https://shop.example.com/"
  interval_sec = 300
  expect_text  = "Our products"
  severity     = "major"
  folder       = "Shop"
  tags         = ["prod"]
}
```

## Exemple complexe — checkout en cinq étapes

Le worker ouvre la page de connexion, saisit les identifiants, valide, et
échoue si le titre du panier manque. Le mot de passe vient d'une variable pour
ne pas rester dans le dépôt.

```terraform
variable "shop_user" {
  type = string
}

variable "shop_password" {
  type      = string
  sensitive = true
}

resource "pathly_scenario" "checkout" {
  name         = "Checkout funnel"
  type         = "browser"
  interval_sec = 60
  severity     = "critical"
  folder       = "Shop"
  tags         = ["prod", "payment"]
  runbook      = "https://wiki.example.com/ops/checkout-unavailable"

  viewport          = "desktop"
  locale            = "fr-FR"
  scenario_timezone = "Europe/Paris"
  click_delay_ms    = 500

  steps = [
    {
      op  = "goto"
      url = "https://shop.example.com/login"
    },
    {
      op       = "fill"
      selector = "input[name=email]"
      value    = var.shop_user
    },
    {
      op       = "fill"
      selector = "input[name=password]"
      value    = var.shop_password
    },
    {
      op       = "click"
      selector = "button[type=submit]"
      text     = "Sign in"
    },
    {
      op   = "assert_text"
      text = "Your cart"
    },
  ]
}

output "checkout_fingerprint" {
  description = "Change si le parcours est édité dans la console."
  value       = pathly_scenario.checkout.scenario_fingerprint
}
```

La première étape doit être `goto` : le worker ouvre une page avant de pouvoir
cliquer ou remplir quoi que ce soit.

## Schéma

Même schéma que la [version anglaise](../../../docs/resources/scenario.md) :
`type` vaut `http` ou `browser`, `steps` porte le parcours (1 à 50 actions),
`scenario_fingerprint` détecte une édition hors Terraform. Les champs
`value`, `username` et `password` sont sensibles.

Opérations acceptées : `goto`, `click`, `hover`, `fill`, `select`, `upload`,
`scroll`, `switch_tab`, `wait`, `wait_for`, `assert_text`, `assert_visible`,
`assert_no_cmp`, `assert_url`, `press`, `wait_for_response`,
`wait_for_networkidle`, `if_visible`, `if_text`, `assert_amount`,
`assert_json_path`, `assert_header`, `http_auth`.

## Attributs laissés vides

Les attributs optionnels sont aussi calculés : absents de la configuration, ils
gardent la valeur choisie par Pathly ou depuis la console. Le plan reste vide.

Le compromis : **retirer une ligne ne revient pas à la valeur par défaut**.
La même règle vaut pour `steps`. Après un import, la liste est vide, et le
prochain apply écrit l'arbre de la configuration par-dessus celui de la console.

## Import

```sh
terraform import pathly_scenario.checkout mon_01H8ZK…
```

Après l'import d'un parcours navigateur, redéclarez les `steps` : l'API ne les
renvoie jamais.
