---
page_title: "pathly_scenario"
description: |-
  Un escenario de monitorización HTTP.
---

> **La versión en inglés es la canónica.** Es la única que lee el Terraform
> Registry y la que prevalece en caso de divergencia con esta traducción:
> [`docs/resources/scenario.md`](../../../docs/resources/scenario.md).

# pathly_scenario

Un escenario de monitorización HTTP.

Los recorridos de navegador no se gestionan aquí: sus pasos llevan credenciales
de inicio de sesión, que un archivo de Terraform y su estado conservarían en
claro. Créelos desde la consola.

## Ejemplo

```terraform
resource "pathly_scenario" "checkout" {
  name         = "Checkout funnel"
  url          = "https://shop.example.com/cart"
  interval_sec = 60

  expected_status = 200
  expect_text     = "Your cart"
  max_latency_ms  = 1500

  severity = "critical"
  folder   = "Shop"
  tags     = ["prod", "payment"]
  runbook  = "https://wiki.example.com/ops/checkout-unavailable"
}
```

## Esquema

### Obligatorios

- `name` (String) — Nombre mostrado en la consola y en las alertas. De 1 a 120
  caracteres.
- `interval_sec` (Number) — Periodo entre dos ejecuciones, en segundos. `0` para
  un escenario gobernado únicamente por `cron`. De 0 a 2 592 000.

### Opcionales

- `url` (String) — Dirección monitorizada, sobre http o https.
- `type` (String) — Únicamente `http`. **Fuerza el reemplazo**: cambiar el tipo
  destruiría el historial del escenario.
- `enabled` (Boolean) — Cuando es false, el escenario existe pero no se ejecuta.
- `method` (String) — `GET` o `HEAD`. Un monitor que hiciera POST o DELETE
  actuaría sobre el sitio en cada ejecución.
- `expected_status` (Number) — Estado HTTP esperado, de 100 a 599.
- `max_latency_ms` (Number) — Por encima de este valor, la ejecución es un fallo
  de rendimiento. De 100 a 600 000.
- `expect_text` (String) — Texto esperado en la respuesta. Un 200 servido por una
  página de error sigue siendo un fallo. 500 caracteres como máximo.
- `regions` (List of String) — Regiones de sonda, 8 como máximo. Cuando está
  vacío, Pathly elige la región por defecto del plan de suscripción.
- `tags` (List of String) — Etiquetas libres, 10 como máximo, usadas para filtrar
  y agrupar.
- `folder` (String) — Carpeta para organizar la consola, 60 caracteres como
  máximo.
- `severity` (String) — `critical`, `major` o `minor`. Gobierna el escalado.
- `runbook` (String) — Instrucciones de guardia, adjuntas a la alerta. 2000
  caracteres como máximo.
- `cron` (String) — Programación cron, además de `interval_sec` o en su lugar.

### Calculados

- `id` (String) — Identificador asignado por Pathly.
- `last_status` (String) — Veredicto de la última ejecución conocida.
- `muted_until` (String) — Fecha límite del silenciado, fijada desde la consola o
  la API. Aquí es de solo lectura: silenciar es un gesto operativo temporal, no
  un estado deseado, y un `apply` no debe reactivar un escenario que se silenció
  la noche anterior.
- `created_at` (String) — Fecha de creación.

## Atributos dejados vacíos

Los atributos opcionales también son calculados: cuando están ausentes de la
configuración, conservan el valor que Pathly eligió en el momento de la
creación, o el fijado desde la consola. El plan permanece por tanto vacío, no
hay idas y venidas en cada `apply`.

La contrapartida: **eliminar una línea no restablece el valor por defecto**.
Pasar de `severity = "critical"` a nada deja el escenario en `critical`. Para
volver al valor por defecto, fije explícitamente el valor que desee.

## Importación

```sh
terraform import pathly_scenario.checkout mon_01H8ZK…
```

El identificador puede leerse en la URL del escenario en la consola. Tras la
importación, `terraform plan` muestra las diferencias entre la configuración y
lo que devuelve la API: mientras no esté vacío, el `apply` modificará el
escenario existente.
