---
page_title: "pathly_sla_target"
description: |-
  Objetivo de disponibilidad medido sobre una ventana deslizante, con un presupuesto de error.
---

> **La versión en inglés es la canónica.** Es la única que lee el Terraform
> Registry y la que prevalece en caso de divergencia con esta traducción:
> [`docs/resources/sla_target.md`](../../../docs/resources/sla_target.md).

# pathly_sla_target

Un objetivo de disponibilidad, medido sobre una ventana deslizante con un
presupuesto de error.

## Ejemplo

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

Con estos valores, el objetivo tolera unos 43 minutos de indisponibilidad por
periodo de 30 días, y avisa cuando se ha consumido el 80 % de ese presupuesto,
es decir, alrededor de 35 minutos, cuando todavía queda margen para actuar.

## Esquema

### Obligatorios

- `objective_pct` (Number) — Disponibilidad perseguida, de 50 a 100. Por ejemplo
  `99.9`.
- `window_days` (Number) — Ventana deslizante de medición, en días, de 1 a 365.

### Opcionales

- `scenario_id` (String) — Escenario medido. Cuando está ausente, el objetivo se
  aplica a toda la organización. **Fuerza el reemplazo**: un objetivo se
  identifica por su escenario, desplazarlo equivaldría a medir otra cosa bajo el
  mismo identificador.
- `name` (String) — Etiqueta mostrada en los informes. 120 caracteres como
  máximo.
- `exclude_maintenance` (Boolean) — Cuando es true, las ventanas de mantenimiento
  no consumen el presupuesto de error.
- `warn_at_budget_ratio` (Number) — Proporción del presupuesto de error consumido
  que desencadena el aviso, de 0.1 a 1.
- `enabled` (Boolean) — Cuando es false, el objetivo sigue definido pero ya no
  avisa.

### Calculados

- `id` (String) — Identificador asignado por Pathly.

## Creación y modificación

La API expone un *upsert*: la creación y la modificación pasan por la misma
llamada, identificada por el escenario objetivo. Un objetivo ya definido en la
consola para ese escenario será por tanto adoptado por el primer `apply`, sin
error de conflicto.

## Importación

Prefiera un bloque `import` (Terraform / OpenTofu ≥ 1.5).

```terraform
import {
  to = pathly_sla_target.checkout
  id = "sla_01H8ZK…"
}
```

```sh
terraform import pathly_sla_target.checkout sla_01H8ZK…
```
