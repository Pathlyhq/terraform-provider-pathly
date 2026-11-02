---
page_title: "pathly_scenarios"
description: |-
  Inventario de los escenarios de la organización, filtrable por etiqueta o por carpeta.
---

> **La versión en inglés es la canónica.** Es la única que lee el Terraform
> Registry y la que prevalece en caso de divergencia con esta traducción:
> [`docs/data-sources/scenarios.md`](../../../docs/data-sources/scenarios.md).

# pathly_scenarios

Lista los escenarios de la organización. Se recorre cada página: una data source
que se detuviera en la primera produciría bucles `for_each` silenciosamente
incompletos.

Dos usos habituales: adoptar un parque existente sin reescribirlo todo, y
detectar los escenarios creados a mano que escapan a las revisiones.

## Ejemplo

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

## Esquema

### Opcionales

- `filter_tag` (String) — Conserva únicamente los escenarios que llevan esta
  etiqueta.
- `filter_folder` (String) — Conserva únicamente los escenarios de esta carpeta.
- `filter_space_id` (String) — Conserva únicamente los escenarios de este
  Espacio.

Los filtros se combinan. `filter_tag` y `filter_folder` se aplican después de la
lectura: no reducen el número de llamadas a la API. `filter_space_id` lo aplica
la API, que responde 404 para un Espacio que no conoce.

### Calculados

- `scenarios` (List of Object) — Escenarios seleccionados, cada uno con `id`,
  `name`, `type`, `url`, `enabled`, `interval_sec`, `folder`, `severity`, `tags`,
  `last_status` y `space_id`.

## Alcance necesario

`scenarios:read`. Un rechazo se manifiesta como un error: devolver una lista
vacía destruiría todo un parque en el siguiente `apply`, si el output alimenta un
`for_each`.
