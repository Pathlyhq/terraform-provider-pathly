---
page_title: "pathly_maintenance_window"
description: |-
  Ventana de mantenimiento, puntual o semanal. Los fallos que ocurren dentro de ella no desencadenan ninguna alerta.
---

> **La versión en inglés es la canónica.** Es la única que lee el Terraform
> Registry y la que prevalece en caso de divergencia con esta traducción:
> [`docs/resources/maintenance_window.md`](../../../docs/resources/maintenance_window.md).

# pathly_maintenance_window

Una ventana durante la cual los fallos no alertan y, si los objetivos de
disponibilidad así lo piden, no consumen el presupuesto de error.

Dos formas, mutuamente excluyentes:

- **semanal** — `weekday`, `start_minute` y `duration_min`,
- **puntual** — `starts_at` y `ends_at`.

Sin `scenario_id`, la ventana cubre toda la organización.

~> **Sin modificación in situ.** La API no puede reescribir una ventana:
cualquier cambio destruye el recurso y lo vuelve a crear. Esto es visible en el
plan.

## Ejemplos

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

## Esquema

### Opcionales

Todos los atributos **fuerzan el reemplazo**.

- `scenario_id` (String) — Escenario objetivo. Cuando está ausente, la ventana
  cubre toda la organización.
- `starts_at` (String) — Inicio, en ISO 8601. Forma puntual.
- `ends_at` (String) — Fin, en ISO 8601. Forma puntual. La API rechaza una
  ventana puntual de más de 90 días: una ventana que se alarga oculta
  interrupciones reales.
- `weekday` (Number) — Día de la semana, 1 para el lunes hasta 7 para el domingo.
  Forma semanal.
- `start_minute` (Number) — Minuto de inicio dentro del día, de 0 a 1439 (hora
  UTC). `180` significa las 3 de la mañana.
- `duration_min` (Number) — Duración en minutos, de 15 a 1440.
- `reason` (String) — Motivo mostrado en la consola y en la traza de auditoría.
  500 caracteres como máximo.

### Calculados

- `id` (String) — Identificador asignado por Pathly.

## Importación

```sh
terraform import pathly_maintenance_window.backup mw_01H8ZK…
```
