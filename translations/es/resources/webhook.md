---
page_title: "pathly_webhook"
description: |-
  Webhook de salida firmado, invocado cuando un escenario falla y cuando se recupera.
---

> **La versión en inglés es la canónica.** Es la única que lee el Terraform
> Registry y la que prevalece en caso de divergencia con esta traducción:
> [`docs/resources/webhook.md`](../../../docs/resources/webhook.md).

# pathly_webhook

Webhook de salida. Cada entrega se firma con HMAC mediante el secreto devuelto en
el momento de la creación: verifique esa firma en el lado receptor, de lo
contrario cualquiera puede falsificar una alerta.

## Ejemplo

```terraform
resource "pathly_webhook" "alerts" {
  url    = "https://hooks.example.com/pathly"
  events = ["run.failed", "run.recovered"]
}
```

## Esquema

### Obligatorios

- `url` (String, sensitive) — Destino, sobre https y público. **Fuerza el
  reemplazo.** Las direcciones privadas, los dominios internos y las IP de
  metadatos de nube se rechazan: el worker llama desde la red de la plataforma, y
  un webhook que apuntara a `169.254.169.254` le haría leer sus propias
  credenciales.

### Opcionales

- `events` (List of String) — `run.failed` y `run.recovered`. Ambos por defecto.
  **Fuerza el reemplazo.**

### Calculados

- `id` (String) — Identificador asignado por Pathly.
- `enabled` (Boolean) — Un webhook desactivado desde la consola permanece en el
  estado, sin ser invocado.
- `url_fingerprint` (String) — Huella del destino. Cambia si la URL se modificó
  en otro sitio: es la única manera de verlo, ya que la API nunca devuelve la URL
  almacenada.
- `secret` (String, sensitive) — Secreto de firma HMAC, devuelto **una sola vez**
  en el momento de la creación.
- `created_at` (String) — Fecha de creación.

## El secreto y el archivo de estado

Dado que el secreto solo se devuelve en el momento de la creación, Terraform
tiene que conservarlo: está por tanto en el estado. Tres consecuencias que hay
que tratar:

1. **Cifre el backend** (`azurerm` con cifrado en reposo, `gcs` con CMEK, o un
   equivalente). Un estado en claro en un bucket público expone el secreto.
2. **No lo publique como un `output` no sensible**: aparecería en los registros
   de la CI.
3. **Para trasladarlo a una bóveda de secretos**, léalo una vez y después elimine
   el output:

```terraform
output "webhook_secret" {
  value     = pathly_webhook.alerts.secret
  sensitive = true
}
```

```sh
terraform output -raw webhook_secret | az keyvault secret set --vault-name … --name pathly-webhook --value @-
```

## Importación

```sh
terraform import pathly_webhook.alerts wh_01H8ZK…
```

~> La importación emite una advertencia: la API no devuelve ni la URL ni el
secreto. Vuelva a indicar `url` en la configuración, de lo contrario el plan
propondría un reemplazo, y recupere el secreto de su bóveda de secretos, puesto
que solo se mostró en el momento de la creación.
