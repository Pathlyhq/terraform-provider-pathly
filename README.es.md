# Pathly Terraform Provider

[English](README.md) · [Français](README.fr.md) · **Español**


[![CI](https://github.com/Pathlyhq/terraform-provider-pathly/actions/workflows/ci.yml/badge.svg)](https://github.com/Pathlyhq/terraform-provider-pathly/actions/workflows/ci.yml)
[![Powered by Pathly](https://img.shields.io/badge/Powered%20by-Pathly-0B5FFF?style=flat-square)](https://pathlyhq.com)
[![Website](https://img.shields.io/badge/Website-pathlyhq.com-111827?style=flat-square)](https://pathlyhq.com)
[![API docs](https://img.shields.io/badge/API-developers-2563eb?style=flat-square)](https://pathlyhq.com/es/developers)
[![Start free](https://img.shields.io/badge/Solo-start%20free-16a34a?style=flat-square)](https://pathlyhq.com/es/login?mode=signup)

**Para usar este provider necesita una clave API. Obtenga su clave gratuita registrándose aquí: [https://pathlyhq.com/es/login?mode=signup&utm_source=github&utm_medium=readme&utm_campaign=signup_cta](https://pathlyhq.com/es/login?mode=signup&utm_source=github&utm_medium=readme&utm_campaign=signup_cta).**

> **Empiece en un clic.** Cree una cuenta gratuita en [Pathly](https://pathlyhq.com) ([registro](https://pathlyhq.com/es/login?mode=signup&utm_source=github&utm_medium=readme&utm_campaign=signup_cta)), genere una clave API en la consola y exporte `PATHLY_API_TOKEN`. Este repositorio es el puente oficial hacia [la monitorización Pathly](https://pathlyhq.com): comprobaciones HTTP y de navegador (carrito, login, disponibilidad), con datos en la UE. Referencia API: [pathlyhq.com/es/developers](https://pathlyhq.com/es/developers).

> **La versión en inglés es la canónica.** Es la única que lee el Terraform
> Registry y la que prevalece en caso de divergencia con esta traducción:
> [`README.md`](README.md).

Gestione la monitorización de Pathly como código: escenarios, ventanas de
mantenimiento, webhooks de salida y objetivos de disponibilidad. Construido con
[terraform-plugin-framework](https://developer.hashicorp.com/terraform/plugin/framework),
sobre la API pública `/v1`, documentada en
[pathlyhq.com/es/developers](https://pathlyhq.com/es/developers).

```hcl
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
  tags         = ["prod", "payment"]
}
```

## Autenticación

La clave de API se suministra mediante la variable de entorno, nunca en un
archivo `.tf`:

```sh
export PATHLY_API_TOKEN="sp_…"
terraform plan
```

| Variable | Función |
|---|---|
| `PATHLY_API_TOKEN` | Clave de API de la organización, prefijo `sp_`. Obligatoria. |
| `PATHLY_API_URL` | Base de la API. Por defecto `https://api.pathlyhq.com`. |

El atributo `api_token` del bloque `provider` existe, pero una clave leída desde
una variable de Terraform acaba **en claro en el estado**. El entorno es la única
vía que no deja rastro, ni en el repositorio ni en el archivo de estado.

### Alcances mínimos de la clave

El provider solo llama a lo que necesita. Cree la clave con los alcances
estrictamente exigidos por los recursos que gestione:

| Recurso gestionado | Alcances |
|---|---|
| `pathly_scenario`, `pathly_scenarios` | `scenarios:read`, `scenarios:write` |
| `pathly_settings` | `org:read`, `org:write` |
| `pathly_webhook` | `alerting:read`, `alerting:write` |
| `pathly_maintenance_window` | `maintenance:read`, `maintenance:write` |
| `pathly_sla_target` | `sla:read`, `sla:write` |
| `pathly_incidents` | `incidents:read` |
| `pathly_members` | `members:read` |
| `pathly_runs`, `pathly_run` | `runs:read` |
| `pathly_usage` | `org:read` |

Para un pipeline que solo ejecuta `terraform plan`, los alcances `:read` bastan.

No se necesita ningún alcance de organización: el provider comprueba la clave en
el momento de la configuración llamando a `/v1/usage`, y **tolera un rechazo** de
esa llamada. Un 403 demuestra que la clave es válida, solo señala la ausencia de
`org:read`. Exigir ese alcance obligaría a una configuración dedicada únicamente
a escenarios a pedir acceso a los ajustes de la organización.

No existe ningún alcance `keys:*` ni ningún alcance `members:write`: una clave de
API no puede crear otra, ni invitar a una cuenta.

## Recursos y data sources

| Nombre | Qué gestiona |
|---|---|
| `pathly_scenario` | Control HTTP o recorrido de navegador (pasos de solo escritura) |
| `pathly_settings` | Ajustes de la organización (singleton, destroy no los restablece) |
| `pathly_maintenance_window` | Ventana de mantenimiento, puntual o semanal |
| `pathly_webhook` | Webhook de salida firmado |
| `pathly_sla_target` | Objetivo de disponibilidad y presupuesto de error |
| `pathly_scenarios` | Inventario de escenarios |
| `pathly_settings` (datos) | Ajustes, solo lectura |
| `pathly_usage` | Consumo del plan |
| `pathly_incidents` | Incidentes |
| `pathly_members` | Miembros, solo lectura |
| `pathly_runs`, `pathly_run` | Ejecuciones |
| `pathly_sla`, `pathly_sla_targets` | Objetivos, con o sin medición |
| `pathly_webhooks` | Webhooks sin su destino |
| `pathly_maintenance_windows` | Ventanas |
| `pathly_status_page` | Página de estado pública |

Documentación por recurso en [`docs/`](docs/) (traducción al español en
[`translations/es/`](translations/es/)), ejemplos completos en
[`examples/`](examples/).

## Lo que el provider no hace, y por qué

- **Ni organizaciones, ni usuarios, ni claves de API.** Una clave capaz de emitir
  claves o de invitar cuentas convierte una fuga de token en una toma de control
  de la organización. Esas acciones se quedan en la consola, con una cuenta
  nominativa y una traza de auditoría.
- **Ni silenciado de incidentes, ni reconocimiento de incidentes.** Son gestos
  operativos temporales, no un estado deseado: ponerlos en el código supondría
  que un `apply` reactive un escenario deliberadamente silenciado la noche
  anterior. `muted_until` se expone en solo lectura.
- **Ningún destinatario de notificaciones individual** (direcciones, números de
  teléfono). Son datos personales: escribirlos en un archivo de estado de
  Terraform, a menudo compartido y rara vez cifrado, crea una obligación RGPD
  que nadie ha pedido.

## Comportamiento cuando las cosas van mal

- **Recurso eliminado desde la consola**: la lectura lo retira del estado, el
  siguiente plan lo vuelve a crear. Un rechazo (403) o una interrupción del
  servicio (5xx) nunca se confunden con una desaparición, de lo contrario un
  `apply` crearía un duplicado junto al objeto existente.
- **Límite de tasa**: el cliente respeta la cabecera `Retry-After`, con un tope
  de 90 segundos y cuatro intentos, y después falla de forma visible.
- **Respuesta perdida tras una creación**: toda creación lleva una clave de
  idempotencia, reutilizada por los reintentos internos. El reintento devuelve el
  recurso ya creado en lugar de crear un segundo, que Terraform desconocería y
  nunca destruiría.
- **Webhook importado**: la API nunca devuelve ni la URL almacenada ni el secreto
  de firma. La importación emite una advertencia y le pide que vuelva a indicar
  `url` en la configuración.

## Desarrollo

```sh
go build ./...                                   # build
go test ./internal/... -coverprofile=covprofile   # tests, 100% of statements
go tool cover -func=covprofile                    # per-function detail
go vet ./...
```

`main.go` no está cubierto: solo contiene la llamada de servicio del plugin, que
es bloqueante. La cobertura se mide sobre `./internal/...`, donde reside todo el
comportamiento.

### Prueba local, sin publicar

```hcl
# ~/.terraformrc
provider_installation {
  dev_overrides {
    "pathlyhq/pathly" = "/path/to/your/GOBIN"
  }
  direct {}
}
```

```sh
go install .   # drops the binary into $GOBIN
cd examples/complete && terraform plan
```

Con un `dev_overrides`, `terraform init` no es necesario y muestra una
advertencia: es lo esperado.

### Contra una API local

```sh
export PATHLY_API_URL="http://localhost:8080"
export PATHLY_API_TOKEN="sp_…"
```

El HTTP en claro solo se tolera en `localhost`: en cualquier otro lugar, la clave
viajaría legible en la cabecera `Authorization`.

## Publicación

Este repositorio es la fuente, y sus workflows de GitHub Actions compilan, firman
y publican los archivos. Hay dos canales. Son independientes, y los mismos
archivos firmados alimentan ambos.

| Canal | Dirección de la source | Quién puede instalarlo |
|---|---|---|
| Terraform Registry público | `pathlyhq/pathly` | cualquiera |
| Registro privado de HCP Terraform | `app.terraform.io/Pathly/pathly` | los miembros de la organización `Pathly` |

El registro público se autentica a través de GitHub, su namespace **es** el
nombre de la organización de GitHub, e ingiere las versiones mediante un webhook
sobre las releases de este repositorio público. Nunca lee una release sin firma:
una release sin `SHA256SUMS.sig` se ignora, y `terraform init` rechaza entonces
el provider.

### Configuración, una sola vez

**1. Clave de firma.** Ambos registros rechazan las curvas elípticas, así que la
clave tiene que ser RSA:

```sh
gpg --full-generate-key            # tipo RSA, 4096 bits, con frase de contraseña
gpg --list-secret-keys --keyid-format=long     # anotar la huella
gpg --armor --export <huella>                  # parte pública, para pegar en el registro
gpg --armor --export-secret-keys <huella>      # parte privada, para el entorno release
```

**2. Entorno `release`** (*Settings → Environments*), limitado a la rama `main`,
para que ninguna otra rama pueda leer la clave:

| Secreto | Contenido |
|---|---|
| `GPG_PRIVATE_KEY` | Parte privada de la clave, en ASCII armor o en base64 |
| `GPG_PASSPHRASE` | Frase de contraseña de esa clave |
| `TFE_TOKEN` | Opcional, registro privado de HCP: token de equipo con *Manage private registry* |
| `TFE_GPG_KEY_ID` | Opcional, registro privado de HCP: id devuelto al registrar la clave pública |

**3. Registro público.** En [registry.terraform.io](https://registry.terraform.io),
con la sesión de GitHub: la clave pública en *User settings → Signing keys*, y
después *Publish → Provider* sobre este repositorio (instala el webhook).

**4. Registro privado de HCP** (opcional). Registrar la clave pública una vez, y
guardar el id devuelto como `TFE_GPG_KEY_ID`:

```sh
curl -sS -X POST "https://app.terraform.io/api/registry/private/v2/gpg-keys" \
  -H "Authorization: Bearer $TFE_TOKEN" \
  -H "Content-Type: application/vnd.api+json" \
  -d "$(jq -n --arg ns Pathly --arg key "$(gpg --armor --export <huella>)" \
        '{data:{type:"gpg-keys",attributes:{namespace:$ns,"ascii-armor":$key}}}')" \
  | jq -r '.data.attributes["key-id"]'
```

Quienes usan el registro privado necesitan un token para el host:
`terraform login app.terraform.io`, o `TF_TOKEN_app_terraform_io` en CI.

### Para cada versión

release-please mantiene al día una pull request `chore(main): release x.y.z` a
partir de los Conventional Commits fusionados en `main`. Mientras la API esté en
v1, las versiones avanzan de un patch en un patch (`feat` incluido). Fusionar esa
pull request es la decisión de publicar:

1. release-please crea una release de GitHub en **borrador**;
2. el job `goreleaser` verifica los módulos Go, ejecuta las pruebas, compila los
   once objetivos, firma las sumas de verificación, certifica su procedencia,
   adjunta los archivos y el manifiesto, y publica la release, lo que crea la etiqueta `vX.Y.Z` y dispara el webhook del registro;
3. `publish-hcp` sube los mismos archivos al registro privado cuando `TFE_TOKEN`
   está definido.

El `terraform init` de los clientes consume de inmediato una versión publicada,
y nunca podrá retirarse de un registro.

### Comprobar que una versión ha llegado realmente

```sh
# Public registry
curl -s https://registry.terraform.io/v1/providers/pathlyhq/pathly/versions | jq '.versions[].version'

# HCP private registry
curl -s -H "Authorization: Bearer $TFE_TOKEN" \
  "https://app.terraform.io/api/v2/organizations/Pathly/registry-providers/private/Pathly/pathly/versions" \
  | jq -r '.data[].attributes.version'
```

En el registro público, que falte una versión mientras la release de GitHub está
publicada casi siempre señala una firma rechazada: un archivo sin
`SHA256SUMS.sig`, o una huella declarada al registro distinta de la que firmó
realmente. En HCP, vuelva a ejecutar `publish-hcp`: las llamadas se pueden
repetir sin riesgo.

### Verificar un archivo

A partir de la versión siguiente a la v0.1.7, cada archivo, el fichero de sumas de
verificación y el manifiesto de una release tienen una atestación de procedencia,
firmada con Sigstore y guardada en este repositorio. Demuestra que el fichero lo
compiló el workflow de release de este repositorio, desde el commit de la release:

```sh
gh release download vX.Y.Z -R Pathlyhq/terraform-provider-pathly -p '*_linux_amd64.zip'
gh attestation verify terraform-provider-pathly_X.Y.Z_linux_amd64.zip -R Pathlyhq/terraform-provider-pathly
```

La v0.1.7 y las versiones anteriores no tienen atestación. Para todas las
versiones, la firma GPG de las sumas de verificación (`SHA256SUMS.sig`) sigue
siendo lo que comprueba `terraform init`. La CI también ejecuta `go mod verify` y
[govulncheck](https://go.dev/doc/security/vuln/) en cada cambio.

## Pathly + Terraform — lista de adopción

1. **Terraform Registry** — `source = "pathlyhq/pathly"`. No vendoring del binario.
2. **Auth** — solo `PATHLY_API_TOKEN` en CI. Nunca la clave en Git.
3. **Import** — `terraform import pathly_scenario.name 3f2b8c1e-6d4a-4b7f-9c2e-1a5d8e7f0b34` para traer monitores creados en la consola.
4. **Docs** — páginas en [`docs/`](docs/) para el Registry (`tfplugindocs`). El inglés es canónico en el Registry.
5. **Consola Pathly** — cree la clave API en [pathlyhq.com](https://pathlyhq.com). [Desarrolladores](https://pathlyhq.com/es/developers).


## Paquetes relacionados

| Package | Role |
|---|---|
| [pathly-opentofu](https://github.com/pathlyhq/pathly-opentofu) | OpenTofu docs & examples |
| [pathly-cdktf](https://github.com/pathlyhq/pathly-cdktf) | CDK for Terraform |
| [pathly-pulumi](https://github.com/pathlyhq/pathly-pulumi) | Pulumi |
| [pathly-ansible](https://github.com/pathlyhq/pathly-ansible) | Ansible |
| [pathly-sdk-go](https://github.com/pathlyhq/pathly-sdk-go) | Go SDK |
| [pathly-sdk-python](https://github.com/pathlyhq/pathly-sdk-python) | Python SDK |
| [Pathly product](https://pathlyhq.com) | [Pathly monitoring](https://pathlyhq.com) |
## Acerca de Pathly

[Pathly](https://pathlyhq.com) es monitorización sintética para agencias y e-commerce: reproduce el recorrido del cliente, detecta un checkout roto antes de la llamada, y deja la prueba (captura, paso, runbook) lista para la factura. Producto: [pathlyhq.com](https://pathlyhq.com) · Desarrolladores: [pathlyhq.com/es/developers](https://pathlyhq.com/es/developers) · Precios: [pathlyhq.com/es/pricing](https://pathlyhq.com/es/pricing).

## Autor

| | |
|---|---|
| **Empresa** | Pathly |
| **Autor** | Simon Raynaud / keyral |

Véase [AUTHORS](AUTHORS).
