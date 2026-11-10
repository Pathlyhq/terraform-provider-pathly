---
page_title: "pathly_spaces"
description: |-
  Espacios activos de la organización, con el límite del plan.
---

> **La versión inglesa es la de referencia.** Esta página traduce
> [`docs/data-sources/spaces.md`](../../../docs/data-sources/spaces.md), la
> única versión que lee el registro de Terraform. Si hay divergencia, prevalece
> la inglesa.

# pathly_spaces

Lista los Espacios activos de la organización, ordenados por nombre, con el
número de Espacios que permite el plan. Una clave limitada a un Espacio solo ve
el suyo.

```terraform
data "pathly_spaces" "all" {}

locals {
  space_by_name = { for s in data.pathly_spaces.all.spaces : s.name => s.id }
}
```

## Esquema

- `max_spaces` (Number) — Espacios activos que permite el plan. Nulo si el plan
  no fija límite.
- `spaces` (List of Object) — Espacios activos, cada uno con `id`, `name`,
  `slug`, `kind`, `status_public`, `scenario_count` y `created_at`. Los
  Espacios archivados no aparecen.

Mientras los Espacios no estén abiertos para la organización, la API responde
404 y la data source falla con un error explícito: una lista vacía se leería
como «ningún cliente» y alimentaría un `for_each` que destruye.

Ámbito necesario: `org:read`.
