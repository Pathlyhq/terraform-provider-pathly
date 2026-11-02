---
page_title: "pathly_account"
description: |-
  Organización y Espacio de la clave, plan activo, cuotas usadas y restantes.
---

> **La versión inglesa es la de referencia.** Esta página traduce
> [`docs/data-sources/account.md`](../../../docs/data-sources/account.md), la
> única versión que lee el registro de Terraform. Si hay divergencia, prevalece
> la inglesa.

# pathly_account

Lee a quién pertenece la clave y el margen que deja el plan: organización,
Espacio de la clave, plan activo, y cada cuota con lo usado y lo restante. Útil
como `precondition` antes de añadir Flow o Espacios que el plan no cubre.

```terraform
data "pathly_account" "this" {}

check "flow_room" {
  assert {
    # Un restante nulo significa que el plan no fija cuota.
    condition     = coalesce(data.pathly_account.this.quotas.flow.remaining, 1) > 0
    error_message = "No quedan Flow en el plan. Ver ${data.pathly_account.this.billing_url}."
  }
}
```

## Esquema

- `organization_id`, `organization_name` — Organización. Su identificador es
  también el de su página de estado, para importarla en `pathly_status_page`.
- `space_id`, `space_name` — Espacio al que está limitada la clave, nulos para
  una clave de toda la organización.
- `plan_id`, `plan_name`, `plan_price_monthly_eur` — Plan activo y su precio
  mensual de catálogo, en euros sin IVA.
- `billing_url` — Pantalla del plan en la app.
- `quotas` — `ping`, `chain`, `flow`, `runs` (con `period` y
  `pack_runs_remaining`), `assistant_trials` (con `period`) y `spaces` (nulo
  para una clave de Espacio, y mientras los Espacios no estén abiertos), cada
  uno con `used`, `limit` y `remaining`. Un `limit` nulo significa sin cuota.

Para una clave limitada a un Espacio, `used` solo cuenta ese Espacio, mientras
que `limit` y `remaining` siguen siendo los de la organización.

Ámbito necesario: `org:read`.
