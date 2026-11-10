---
page_title: "pathly_status_page"
description: |-
  La página de estado de la organización, o la de un Espacio, con su insignia.
---

> **La versión inglesa es la de referencia.** Esta página traduce
> [`docs/resources/status_page.md`](../../../docs/resources/status_page.md),
> la única versión que lee el registro de Terraform. Si hay divergencia,
> prevalece la inglesa.

# pathly_status_page

La página de estado de la organización (sin `space_id`) o la de un Espacio
(con `space_id`), con la dirección de su insignia SVG.

La creación adopta una página que ya existe: la API la devuelve tal cual y
aplica `public`. Un primer apply adopta así la página creada en la consola,
sin import.

## Ejemplo

```terraform
resource "pathly_status_page" "shop" {
  space_id = pathly_space.shop.id
  public   = true
}

output "shop_badge" {
  value = pathly_status_page.shop.badge_url
}
```

## Esquema

- `space_id` (String, opcional) — Espacio al que pertenece la página. Ausente
  para la página de la organización. **Fuerza el reemplazo.**
- `slug` (String, opcional) — Dirección de la página de la organización,
  propuesta a partir de su nombre si falta. **Cambiarla rompe los enlaces e
  insignias ya colocados.** Rechazado para una página de Espacio, que toma la
  dirección de su Espacio.
- `public` (Boolean, opcional) — Publica la página. Una página nueva queda
  privada salvo `true`. Publicar expone los nombres de los escenarios y su
  disponibilidad.
- Calculados: `id` (el de la organización o el del Espacio), `kind`
  (`organization` o `space`), `name`, `url` y `badge_url` (añada `?v=uptime`
  para la disponibilidad). `url` y `badge_url` conservan su valor en el plan
  mientras `slug` no cambie.

## Destruir una página de estado la despublica

La API no tiene borrado para una página de estado. Un destroy **despublica**
la página (`public = false`), conserva su dirección y emite una advertencia.
Terraform deja de seguirla. Publicarla de nuevo más tarde recupera las
insignias ya pegadas en los sitios de los clientes.

## Página de la organización y `pathly_settings`

`status_slug` y `status_public` de `pathly_settings` controlan la misma página
de la organización. Gestiónela en un solo lugar. Las páginas de Espacio solo se
gestionan aquí.

## Import

El identificador de la página de la organización es el de la organización
(`data.pathly_account.this.organization_id`). El de una página de Espacio es el
del Espacio.

```sh
terraform import pathly_status_page.shop 6f1c2a9e-3b7d-4c58-a0e4-9d2f5b8c1a67
```

## Ámbitos necesarios

`status:read` para refrescar, `status:write` para crear, publicar, despublicar
y mover.
