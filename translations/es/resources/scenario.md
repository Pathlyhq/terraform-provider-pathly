---
page_title: "pathly_scenario"
description: |-
  Un control HTTP o un recorrido de navegador.
---

> **La versión inglesa es la de referencia.** Esta página traduce
> [`docs/resources/scenario.md`](../../../docs/resources/scenario.md), la única
> versión que lee el registro de Terraform. Si hay divergencia, prevalece la
> inglesa. Una traducción caducada que anunciara ámbitos de API incorrectos
> llevaría a crear una clave con demasiados privilegios.

# pathly_scenario

Un control HTTP o un recorrido de navegador.

Un recorrido de navegador es de solo escritura: la API nunca devuelve los
pasos, solo `scenario_fingerprint`. Un paso `fill` o `http_auth` puede llevar
una contraseña — guarde esos valores en un almacén de secretos, no en el
repositorio. El estado los conserva, marcados como sensibles.

## Ejemplo simple — control HTTP

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

## Ejemplo complejo — checkout en cinco pasos

El worker abre la página de inicio de sesión, escribe las credenciales, envía
el formulario y falla si falta el título del carrito. La contraseña viene de
una variable para no quedar en el repositorio.

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
  description = "Cambia si el recorrido se edita en la consola."
  value       = pathly_scenario.checkout.scenario_fingerprint
}
```

El primer paso debe ser `goto`: el worker abre una página antes de poder
hacer clic o rellenar nada.

## Esquema

El mismo esquema que la [versión inglesa](../../../docs/resources/scenario.md):
`type` vale `http` o `browser`, `steps` lleva el recorrido (1 a 50 acciones),
`scenario_fingerprint` detecta una edición fuera de Terraform. `value`,
`username` y `password` son sensibles.

## Atributos dejados vacíos

Los atributos opcionales también son calculados: si faltan en la
configuración, conservan el valor que Pathly eligió o el de la consola.

La contrapartida: **quitar una línea no vuelve al valor por defecto**. La
misma regla vale para `steps`. Tras un import la lista está vacía, y el
siguiente apply escribe el árbol de la configuración encima del de la consola.

## Import

Prefiera un bloque `import` (Terraform / OpenTofu ≥ 1.5); la forma CLI es
equivalente.

```terraform
import {
  to = pathly_scenario.checkout
  id = "mon_01H8ZK…"
}
```

```sh
terraform import pathly_scenario.checkout mon_01H8ZK…
```

Tras importar un recorrido de navegador, vuelva a declarar `steps`: la API
nunca los devuelve. Ejemplo completo: `examples/import` (inglés canónico).
