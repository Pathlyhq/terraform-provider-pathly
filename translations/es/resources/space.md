---
page_title: "pathly_space"
description: |-
  Un Espacio: los escenarios de un cliente, una marca o una aplicación.
---

> **La versión inglesa es la de referencia.** Esta página traduce
> [`docs/resources/space.md`](../../../docs/resources/space.md), la única
> versión que lee el registro de Terraform. Si hay divergencia, prevalece la
> inglesa.

# pathly_space

Un Espacio agrupa los escenarios de un cliente, una marca o una aplicación.
Una agencia describe un Espacio por cliente, coloca en él los sitios del
cliente con `space_id` en `pathly_scenario`, y publica una página de estado por
cliente con `pathly_status_page`.

Los Espacios deben estar abiertos para la organización. Hasta entonces la API
responde 404 en todas las rutas de los Espacios: una creación falla con ese
error, y un Espacio que ya está en el estado sale de él en el siguiente
refresco.

## Ejemplo

```terraform
resource "pathly_space" "shop" {
  name = "Example shop"
  kind = "client"
}

resource "pathly_scenario" "shop_home" {
  name         = "Example shop — página de inicio"
  url          = "https://shop.example.com"
  interval_sec = 300
  space_id     = pathly_space.shop.id
}
```

Ejemplo completo: [`examples/agency`](../../../examples/agency/).

## Esquema

- `name` (String, obligatorio) — Nombre del Espacio, de 1 a 80 caracteres,
  único entre los Espacios activos de la organización.
- `kind` (String, opcional) — `client`, `site` o `application`. Si falta, Pathly
  lo deduce del perfil de uso de la organización.
- Calculados: `id`, `slug` (dirección del Espacio, conservada al renombrar),
  `status_public` (página del Espacio publicada o no, gestionada por
  `pathly_status_page`) y `created_at`.

## Destruir un Espacio lo archiva

La API no tiene borrado para un Espacio: `DELETE /v1/spaces/{id}` lo archiva.
Sus escenarios salen de él y **siguen vigilados**, no se borra nada, y el
nombre vuelve a quedar libre. Para dejar de vigilar a un cliente, destruya
también sus recursos `pathly_scenario`.

## Import

```sh
terraform import pathly_space.shop 6f1c2a9e-3b7d-4c58-a0e4-9d2f5b8c1a67
```

La data source `pathly_spaces` lista los identificadores.

## Ámbitos necesarios

`org:read` para refrescar, `org:write` para crear, renombrar y archivar. Una
clave limitada a un Espacio no puede crear ninguno.
