---
page_title: "pathly_settings"
description: |-
  Ajustes de la organización: alertas, página de estado, zona horaria y triaje.
---

> **La versión inglesa es la de referencia.** Esta página traduce
> [`docs/resources/settings.md`](../../../docs/resources/settings.md), la única
> versión que lee el registro de Terraform.

# pathly_settings

Ajustes de la organización: alertas, escalado, página de estado, zona horaria
y triaje.

Hay un solo objeto por organización. Declararlo dos veces haría que ambas
definiciones se pisaran en cada apply. Create y Update son la misma llamada:
los ajustes existen antes de Terraform.

`terraform destroy` **no los restablece**. No hay endpoint de borrado, y no
debe haberlo: destruir un workspace de prueba cortaría en silencio las alertas
de toda la organización.

## Ejemplo simple

```terraform
resource "pathly_settings" "this" {
  timezone    = "Europe/Paris"
  alert_email = "ops@example.com"
}
```

## Ejemplo complejo

```terraform
resource "pathly_settings" "this" {
  timezone               = "Europe/Paris"
  alert_email            = "ops@example.com"
  alert_on_recovery      = true
  escalation_after_fails = 3
  escalation_email       = "oncall@example.com"
  weekly_digest_enabled  = true
  weekly_digest_email    = "ops@example.com"

  status_slug   = "acme"
  status_public = true

  ssl_warn_days    = [30, 14, 7]
  domain_warn_days = [30]

  triage_enabled         = true
  triage_confirm_enabled = true
  triage_latency_factor  = 2.5
}
```

Activar `status_public` publica los nombres de los escenarios y su
disponibilidad. Compruebe que ninguno nombra un host interno.

El esquema detallado está en la [versión inglesa](../../../docs/resources/settings.md).
Ámbitos: `org:read` para refrescar, `org:write` para cambiar.

```sh
terraform import pathly_settings.this settings
```
