# Pathly Terraform Provider

[English](README.md) · [Français](README.fr.md) · **Español**


[![CI](https://gitlab.com/pathlyhq/pathly-terraform-provider/badges/main/pipeline.svg)](https://gitlab.com/pathlyhq/pathly-terraform-provider/-/pipelines)
[![Powered by Pathly](https://img.shields.io/badge/Powered%20by-Pathly-0B5FFF?style=flat-square)](https://pathlyhq.com)
[![Website](https://img.shields.io/badge/Website-pathlyhq.com-111827?style=flat-square)](https://pathlyhq.com)
[![API docs](https://img.shields.io/badge/API-developers-2563eb?style=flat-square)](https://pathlyhq.com/es/developers)
[![Start free](https://img.shields.io/badge/Solo-start%20free-16a34a?style=flat-square)](https://pathlyhq.com/es/login?mode=signup)

> **Empiece en un clic.** Cree una cuenta gratuita en [Pathly](https://pathlyhq.com) ([registro](https://pathlyhq.com/es/login?mode=signup)), genere una clave API en la consola y exporte `PATHLY_API_TOKEN`. Este repositorio es el puente oficial hacia [la monitorización Pathly](https://pathlyhq.com): comprobaciones HTTP y de navegador (carrito, login, disponibilidad), con datos en la UE. Referencia API: [pathlyhq.com/es/developers](https://pathlyhq.com/es/developers).

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

El desarrollo vive en GitLab, y el pipeline compila y firma los archivos desde
allí. A dónde van esos archivos es una decisión aparte, y hay dos canales. Son
independientes: cada uno funciona por sí solo, y los mismos archivos firmados
alimentan ambos.

| Canal | Dirección de la source | Quién puede instalarlo | GitHub necesario |
|---|---|---|---|
| Terraform Registry público | `pathlyhq/pathly` | cualquiera | sí, un espejo público |
| Registro privado de HCP Terraform | `app.terraform.io/Pathly/pathly` | los miembros de la organización `Pathly` | no |

El registro público es el único que aporta capacidad de descubrimiento, un sitio
de documentación renderizado y la instalación sin credenciales. Llega también con
una exigencia inflexible: se autentica a través de GitHub, su namespace **es** un
nombre de organización de GitHub, e ingiere las versiones mediante un webhook
sobre las releases de un repositorio público de GitHub. No existe ninguna
importación desde GitLab, ninguna carga manual ni ninguna API para enviar una
versión. GitHub sirve por tanto de escaparate en solo lectura, alimentado por un
espejo, mientras que la clave de firma nunca sale de GitLab.

### Configuración compartida

**1. Clave de firma.** Ambos registros rechazan las curvas elípticas, así que la
clave tiene que ser RSA:

```sh
gpg --full-generate-key            # RSA type, 4096 bits, with a passphrase
gpg --list-secret-keys --keyid-format=long     # note the fingerprint
gpg --armor --export <fingerprint>             # public part, to paste into the registry
gpg --armor --export-secret-keys <fingerprint> | base64 -w0   # private part, for CI
```

**2. Variables CI/CD de GitLab**, todas **masked y protected** (enmascaradas y
protegidas). «Protected» no es una comodidad: sin ese atributo, la clave de firma
es legible desde cualquier rama y, por tanto, exfiltrable mediante un simple
push.

| Variable | Contenido |
|---|---|
| `GPG_PRIVATE_KEY` | Parte privada de la clave, codificada en base64 |
| `GPG_PASSPHRASE` | Frase de contraseña de esa clave |
| `GPG_FINGERPRINT` | Huella de la clave |

### Canal A — Terraform Registry público

El nombre del repositorio de GitHub no se elige libremente: la source
`pathlyhq/pathly` exige una organización `pathlyhq` y un repositorio
`terraform-provider-pathly`.

**1. Repositorio de GitHub.** Cree la organización `pathlyhq` y el repositorio
**público** `terraform-provider-pathly`, vacío, sin README generado y sin
licencia generada. El espejo se negaría a hacer push sobre un historial
divergente.

**2. Espejo GitLab → GitHub.** En **Settings → Repository → Mirroring
repositories** (Ajustes → Repositorio → Duplicación de repositorios), dirección
*Push*, URL `https://github.com/pathlyhq/terraform-provider-pathly.git`, con un
token de GitHub como contraseña. Deje «Mirror only protected branches» (Duplicar
solo las ramas protegidas) **sin marcar**: sin las etiquetas Git, el registro no
tiene nada que leer.

**3. Variable CI/CD** `GITHUB_TOKEN`, masked y protected: un token
**fine-grained** (de grano fino) con el permiso `contents: write` únicamente
sobre el repositorio `terraform-provider-pathly`, y con caducidad corta. Un token
clásico daría acceso a toda la organización.

**4. Declaración al registro.** Inicie sesión en
[registry.terraform.io](https://registry.terraform.io) con la cuenta de GitHub,
declare la clave pública en *User settings → Signing keys* (Ajustes de usuario →
Claves de firma), después *Publish → Provider* (Publicar → Provider) y elija el
repositorio. El webhook se instala en ese momento.

### Canal B — registro privado de HCP Terraform

Ningún GitHub en esta vía. Un provider se publica directamente en el registro de
la organización a través de su API — que además es la única manera, ya que la
consola de HCP y la conexión VCS solo gestionan módulos, nunca providers.

**1. Variable CI/CD** `TFE_TOKEN`, masked y protected: un token de HCP Terraform
perteneciente a un equipo que posea el permiso *Manage private registry*
(Gestionar el registro privado). Prefiera un token de equipo antes que uno
personal, para que el pipeline no deje de funcionar el día en que su autor se
marche.

**2. Registre la clave pública** una sola vez, y conserve el id que devuelve.
Para un registro privado, el namespace es el nombre de la organización:

```sh
curl -sS -X POST "https://app.terraform.io/api/registry/private/v2/gpg-keys" \
  -H "Authorization: Bearer $TFE_TOKEN" \
  -H "Content-Type: application/vnd.api+json" \
  -d "$(jq -n --arg ns Pathly --arg key "$(gpg --armor --export <fingerprint>)" \
        '{data:{type:"gpg-keys",attributes:{namespace:$ns,"ascii-armor":$key}}}')" \
  | jq -r '.data.attributes["key-id"]'
```

**3. Variable CI/CD** `TFE_GPG_KEY_ID` con ese id. Es lo que vincula una versión
publicada con la clave que la firmó.

**4. Los consumidores** necesitan un token para el host, ya que un registro
privado está autenticado. `terraform login app.terraform.io` escribe uno, o
defina `TF_TOKEN_app_terraform_io` en CI:

```hcl
terraform {
  required_providers {
    pathly = {
      source  = "app.terraform.io/Pathly/pathly"
      version = "~> 0.1"
    }
  }
}
```

### Para cada versión

```sh
git tag v0.1.0 && git push origin v0.1.0
```

El pipeline ejecuta las pruebas, valida los ejemplos, y después se detiene.
Siguen tres jobs manuales, en este orden: `archives` compila los once objetivos y
firma las sumas de verificación, tras lo cual `publish-github` y `publish-hcp`
suben esos mismos archivos a los canales que utilice. Ninguno de los dos jobs de
publicación necesita al otro.

`publish-github` crea la release de GitHub en estado de **borrador**, de modo que
el registro no ingiere nada mientras no la publique a mano. Dos gestos
explícitos, y es deliberado: una versión publicada la consume de inmediato el
`terraform init` de los clientes, y nunca podrá retirarse de un registro.

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
realmente.

En HCP, una versión cuyos archivos de sumas de verificación o binarios de
plataforma no se hayan subido todos permanece en su sitio pero inutilizable, y
`terraform init` la reporta como no disponible en lugar de ausente. Vuelva a
ejecutar `publish-hcp` sobre la etiqueta: las llamadas se pueden repetir sin
riesgo.

## Pathly + Terraform — lista de adopción

1. **Terraform Registry** — `source = "pathlyhq/pathly"`. No vendoring del binario.
2. **Auth** — solo `PATHLY_API_TOKEN` en CI. Nunca la clave en Git.
3. **Import** — `terraform import pathly_scenario.name mon_…` para traer monitores creados en la consola.
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
