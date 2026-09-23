---
page_title: "Pathly Provider"
description: |-
  Gestione la monitorización de Pathly como código: escenarios, ventanas de mantenimiento, webhooks y objetivos de disponibilidad.
---

> **La versión en inglés es la canónica.** Es la única que lee el Terraform
> Registry y la que prevalece en caso de divergencia con esta traducción:
> [`docs/index.md`](../../docs/index.md).

# Pathly Provider

Describa la monitorización de Pathly igual que el resto de su infraestructura:
revisada en una *pull request*, aplicada por la CI, idéntica de un entorno a
otro.

```terraform
terraform {
  required_providers {
    pathly = {
      source  = "pathlyhq/pathly"
      version = "~> 0.1"
    }
  }
}

provider "pathly" {}

resource "pathly_scenario" "checkout" {
  name         = "Checkout"
  url          = "https://shop.example.com/cart"
  interval_sec = 300
  expect_text  = "Your cart"
  severity     = "critical"
}
```

## Autenticación

```sh
export PATHLY_API_TOKEN="sp_…"
```

La clave se crea en **Settings → API keys** (Ajustes → Claves de API), desde una
cuenta de propietario o de administrador. Asígnele los alcances estrictamente
necesarios:

| Recursos gestionados | Alcances |
|---|---|
| `pathly_scenario`, `pathly_scenarios` | `scenarios:read`, `scenarios:write` |
| `pathly_webhook` | `alerting:read`, `alerting:write` |
| `pathly_maintenance_window` | `maintenance:read`, `maintenance:write` |
| `pathly_sla_target` | `sla:read`, `sla:write` |

Para un pipeline que solo ejecuta `terraform plan`, los alcances `:read` bastan.
No se exige ningún alcance de organización: la comprobación realizada en el
momento de la configuración tolera su rechazo.

## Esquema

### Opcionales

- `api_token` (String, sensitive) — Clave de API de la organización, prefijo
  `sp_`. Es preferible suministrarla mediante `PATHLY_API_TOKEN`: escrita en un
  archivo `.tf` acaba en el repositorio, y leída desde una variable de Terraform
  acaba en claro en el archivo de estado.
- `api_url` (String) — Base de la API. Por defecto `https://api.pathlyhq.com`.
  El HTTP en claro solo se acepta en `localhost`: en cualquier otro lugar, la
  clave viajaría legible en la cabecera `Authorization`.

Las variables de entorno `PATHLY_API_TOKEN` y `PATHLY_API_URL` tienen prioridad
sobre los atributos del bloque.

## Comportamiento

- **Configuración**: se realiza una llamada de verificación, una sola vez. Una
  clave caducada, revocada o truncada falla aquí, con un mensaje que indica qué
  corregir, en lugar de un error por recurso durante el plan. Una clave válida
  pero con alcances limitados pasa: el rechazo de esa llamada no es un error.
- **Límite de tasa**: se respeta la cabecera `Retry-After`, con un tope de 90
  segundos y cuatro intentos.
- **Creaciones**: toda creación lleva una clave de idempotencia. Si la respuesta
  se pierde, el reintento devuelve el recurso ya creado en lugar de una segunda
  copia.
- **Recurso desaparecido**: sale del estado y el siguiente plan lo vuelve a
  crear. Un rechazo o una interrupción del servicio nunca se confunden con una
  desaparición.

## Fuera de alcance, deliberadamente

El provider no gestiona organizaciones, ni usuarios, ni claves de API, ni
destinatarios de notificaciones, ni el silenciado de incidentes. Los tres
primeros convertirían una fuga de token en una toma de control de la
organización, el cuarto pondría datos personales en un archivo de estado, y el
quinto supondría que un `apply` reactive un escenario silenciado
deliberadamente.
