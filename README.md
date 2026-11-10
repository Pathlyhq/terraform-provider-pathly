# Pathly Terraform Provider

**English** · [Français](README.fr.md) · [Español](README.es.md)


[![CI](https://github.com/Pathlyhq/terraform-provider-pathly/actions/workflows/ci.yml/badge.svg)](https://github.com/Pathlyhq/terraform-provider-pathly/actions/workflows/ci.yml)
[![Powered by Pathly](https://img.shields.io/badge/Powered%20by-Pathly-0B5FFF?style=flat-square)](https://pathlyhq.com)
[![Website](https://img.shields.io/badge/Website-pathlyhq.com-111827?style=flat-square)](https://pathlyhq.com)
[![API docs](https://img.shields.io/badge/API-developers-2563eb?style=flat-square)](https://pathlyhq.com/en/developers)
[![Start free](https://img.shields.io/badge/Solo-start%20free-16a34a?style=flat-square)](https://pathlyhq.com/en/login?mode=signup)

**To use this provider, you need an API key. Get your free API key by signing up here: [https://pathlyhq.com/en/login?mode=signup&utm_source=github&utm_medium=readme&utm_campaign=signup_cta](https://pathlyhq.com/en/login?mode=signup&utm_source=github&utm_medium=readme&utm_campaign=signup_cta).**

> **Get started in one click.** Create a free account on [Pathly](https://pathlyhq.com) ([sign up](https://pathlyhq.com/en/login?mode=signup&utm_source=github&utm_medium=readme&utm_campaign=signup_cta)), create an API key in the console, then export `PATHLY_API_TOKEN`. This project is the official bridge to [Pathly monitoring](https://pathlyhq.com) — real-browser and HTTP checks for checkout, login and availability, with data hosted in the EU. Full API reference: [pathlyhq.com/en/developers](https://pathlyhq.com/en/developers).

> English is the default / canonical documentation language (Terraform Registry reads this file).

Manage Pathly monitoring as code: scenarios, maintenance windows, outbound
webhooks and SLA targets. Built with
[terraform-plugin-framework](https://developer.hashicorp.com/terraform/plugin/framework),
on top of the public `/v1` API, documented at
[pathlyhq.com/en/developers](https://pathlyhq.com/en/developers).

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

## Authentication

The API key is supplied through the environment variable, never in a `.tf`
file:

```sh
export PATHLY_API_TOKEN="sp_…"
terraform plan
```

| Variable | Purpose |
|---|---|
| `PATHLY_API_TOKEN` | Organization API key, `sp_` prefix. Required. |
| `PATHLY_API_URL` | API base. Defaults to `https://api.pathlyhq.com`. |

The `api_token` attribute of the `provider` block does exist, but a key read
from a Terraform variable ends up **in plaintext in the state**. The environment
is the only path that leaves no trace, neither in the repository nor in the
state file.

### Minimum scopes for the key

The provider only calls what it needs. Create the key with the scopes strictly
required by the resources you manage:

| Managed resource | Scopes |
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
| `pathly_space`, `pathly_spaces`, `pathly_account` | `org:read`, `org:write` (`org:read` alone for the two data sources) |
| `pathly_status_page` (resource) | `status:read`, `status:write` |

For a pipeline that only runs `terraform plan`, the `:read` scopes are enough.

No organization scope is needed: the provider checks the key at configuration
time by calling `/v1/usage`, and **tolerates a refusal** of that call. A 403
proves the key is valid, it only signals the absence of `org:read`. Requiring
that scope would force a scenario-only configuration to ask for access to the
organization settings.

There is no `keys:*` scope and no `members:write` scope: an API key cannot
create another one, nor invite an account.

## Resources and data sources

| Name | What it manages |
|---|---|
| `pathly_scenario` | HTTP check or browser journey (write-only steps) |
| `pathly_settings` | Organization settings (singleton, destroy does not reset) |
| `pathly_maintenance_window` | Maintenance window, one-off or weekly |
| `pathly_webhook` | Signed outbound webhook |
| `pathly_sla_target` | SLA target and error budget |
| `pathly_scenarios` | Inventory of scenarios |
| `pathly_settings` (data) | Settings, read-only |
| `pathly_usage` | Plan consumption |
| `pathly_incidents` | Incidents |
| `pathly_members` | Members, read-only |
| `pathly_runs`, `pathly_run` | Executions |
| `pathly_sla`, `pathly_sla_targets` | Objectives, with or without measurement |
| `pathly_webhooks` | Webhooks without their destination |
| `pathly_maintenance_windows` | Windows |
| `pathly_status_page` (data) | Components of the public status page |
| `pathly_space` | Space: the scenarios of one client, brand or application (destroy archives) |
| `pathly_status_page` | Status page of the organization or of a Space, with its badge (destroy unpublishes) |
| `pathly_spaces` | Active Spaces and the limit of the plan |
| `pathly_account` | Organization and Space of the key, plan, quotas used and remaining |

Per-resource documentation in [`docs/`](docs/), complete examples in
[`examples/`](examples/) (including [`examples/import`](examples/import/) for
`import` blocks). The same reference pages are translated in
[`translations/fr/`](translations/fr/) and [`translations/es/`](translations/es/).
English stays canonical: it is the only version the Terraform Registry renders,
so it is the one that prevails when a translation lags behind.

Contributing (tests, `TF_ACC`, no secrets): [`CONTRIBUTING.md`](CONTRIBUTING.md).
Discovery / registry publish checklist: [`docs/SEO.md`](docs/SEO.md).


## What the provider does not do, and why

- **No organizations, no users, no API keys.** A key that can mint keys or
  invite accounts turns a token leak into a takeover of the organization. Those
  actions stay in the console, with a named account and an audit trail.
- **No incident muting, no incident acknowledgement.** These are temporary
  operational gestures, not desired state: putting them in code would mean an
  `apply` wakes up a scenario that was deliberately muted the night before.
  `muted_until` is exposed read-only.
- **No individual notification recipients** (addresses, phone numbers).
  These are personal data: writing them into a Terraform state file, often
  shared and rarely encrypted, creates a GDPR obligation that nobody asked for.

## Behavior when things go wrong

- **Resource deleted from the console**: the read removes it from the state, the
  next plan recreates it. A refusal (403) or an outage (5xx) is never mistaken
  for a disappearance, otherwise an `apply` would create a duplicate next to the
  existing object.
- **Rate limit**: the client honors the `Retry-After` header, with a cap of 90
  seconds and four attempts, then fails visibly.
- **Response lost after a creation**: every creation carries an idempotency key,
  reused by the internal retries. The retry returns the resource already created
  instead of creating a second one, which Terraform would not know about and
  would never destroy.
- **Imported webhook**: the API never returns the stored URL nor the signing
  secret. The import emits a warning and asks you to state `url` again in the
  configuration.

## Development

```sh
go build ./...                                   # build
go test ./internal/... -coverprofile=covprofile   # tests, 100% of statements
go tool cover -func=covprofile                    # per-function detail
go vet ./...
```

`main.go` is not covered: it only contains the plugin serving call, which
blocks. Coverage is measured on `./internal/...`, where all the behavior lives.

### Local trial, without publishing

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

With a `dev_overrides`, `terraform init` is not needed and prints a warning:
that is expected.

### Against a local API

```sh
export PATHLY_API_URL="http://localhost:8080"
export PATHLY_API_TOKEN="sp_…"
```

Plaintext HTTP is tolerated on `localhost` only: anywhere else, the key would
travel readable in the `Authorization` header.

## Publishing

This repository is the source, and its GitHub Actions workflows build, sign and
publish the archives. There are two channels. They are independent, and the same
signed archives feed both.

| Channel | Source address | Who can install it |
|---|---|---|
| Public Terraform Registry | `pathlyhq/pathly` | anyone |
| HCP Terraform private registry | `app.terraform.io/Pathly/pathly` | members of the `Pathly` organization |

The public registry authenticates through GitHub, its namespace **is** the GitHub
organization name, and it ingests versions through a webhook on the releases of
this public repository. It never reads an unsigned release: a release without
`SHA256SUMS.sig` is ignored, and `terraform init` then refuses the provider.

### Setup, once

**1. Signing key.** Both registries reject elliptic curves, so the key has to be
RSA:

```sh
gpg --full-generate-key            # RSA type, 4096 bits, with a passphrase
gpg --list-secret-keys --keyid-format=long     # note the fingerprint
gpg --armor --export <fingerprint>             # public part, to paste into the registry
gpg --armor --export-secret-keys <fingerprint> # private part, for the release environment
```

**2. `release` environment** (*Settings → Environments*), limited to the `main`
branch, so that no other branch can read the key:

| Secret | Contents |
|---|---|
| `GPG_PRIVATE_KEY` | Private part of the key, ASCII armored or base64 |
| `GPG_PASSPHRASE` | Passphrase of that key |
| `TFE_TOKEN` | Optional, HCP private registry: team token with *Manage private registry* |
| `TFE_GPG_KEY_ID` | Optional, HCP private registry: id returned when the public key was registered |

**3. Public registry.** On [registry.terraform.io](https://registry.terraform.io),
signed in with GitHub: the public key under *User settings → Signing keys*, then
*Publish → Provider* on this repository (it installs the webhook).

**4. HCP private registry** (optional). Register the public key once, and store
the id it returns as `TFE_GPG_KEY_ID`:

```sh
curl -sS -X POST "https://app.terraform.io/api/registry/private/v2/gpg-keys" \
  -H "Authorization: Bearer $TFE_TOKEN" \
  -H "Content-Type: application/vnd.api+json" \
  -d "$(jq -n --arg ns Pathly --arg key "$(gpg --armor --export <fingerprint>)" \
        '{data:{type:"gpg-keys",attributes:{namespace:$ns,"ascii-armor":$key}}}')" \
  | jq -r '.data.attributes["key-id"]'
```

Consumers of the private registry need a token for the host:
`terraform login app.terraform.io`, or `TF_TOKEN_app_terraform_io` in CI.

### For each version

release-please keeps a `chore(main): release x.y.z` pull request up to date from
the Conventional Commits merged on `main`. While the API is v1, versions move one
patch at a time (`feat` included). Merging that pull request is the publication
decision:

1. release-please creates a **draft** GitHub release;
2. the `goreleaser` job verifies the Go modules, runs the tests, builds the
   eleven targets, signs the checksums, attests their build provenance, attaches
   the archives and the manifest, then publishes the release,
   which creates the `vX.Y.Z` tag and triggers the registry webhook;
3. `publish-hcp` pushes the same archives to the private registry when
   `TFE_TOKEN` is set.

A published version is consumed at once by `terraform init`, and can never be
withdrawn from a registry.

### Checking that a version really landed

```sh
# Public registry
curl -s https://registry.terraform.io/v1/providers/pathlyhq/pathly/versions | jq '.versions[].version'

# HCP private registry
curl -s -H "Authorization: Bearer $TFE_TOKEN" \
  "https://app.terraform.io/api/v2/organizations/Pathly/registry-providers/private/Pathly/pathly/versions" \
  | jq -r '.data[].attributes.version'
```

On the public registry, a version missing while the GitHub release is published
almost always signals a rejected signature: an archive without `SHA256SUMS.sig`,
or a fingerprint declared to the registry that differs from the one that actually
signed. On HCP, re-run `publish-hcp`: the calls are safe to repeat.

### Verifying an archive

From the version after v0.1.7, each archive, the checksums file and the manifest
of a release come with a build provenance attestation, signed through Sigstore and
stored on this repository. It proves that the file was built by the release
workflow of this repository, from the commit of the release:

```sh
gh release download vX.Y.Z -R Pathlyhq/terraform-provider-pathly -p '*_linux_amd64.zip'
gh attestation verify terraform-provider-pathly_X.Y.Z_linux_amd64.zip -R Pathlyhq/terraform-provider-pathly
```

v0.1.7 and earlier versions have no attestation. For every version, the GPG
signature of the checksums (`SHA256SUMS.sig`) is what `terraform init` checks.
CI also runs `go mod verify` and [govulncheck](https://go.dev/doc/security/vuln/)
on every change.

## Pathly + Terraform — adoption checklist

1. **Terraform Registry** — install with `source = "pathlyhq/pathly"` (public registry). Do not vendor the binary in projects.
2. **Auth** — only `PATHLY_API_TOKEN` in CI (GitHub Actions, GitLab CI, Spacelift, Env0, Scalr). Never commit the key.
3. **Import existing resources** — use `import` blocks or `terraform import pathly_scenario.name 3f2b8c1e-6d4a-4b7f-9c2e-1a5d8e7f0b34` (the identifier is a UUID; webhooks, maintenance windows, SLA targets, settings). See [`examples/import`](examples/import/).
4. **Docs** — resource pages under [`docs/`](docs/) are generated for the Registry (`tfplugindocs`). English is canonical on the Registry. Discovery / SEO notes: [`docs/SEO.md`](docs/SEO.md).
5. **In the Pathly console** — when you create an API key on [pathlyhq.com](https://pathlyhq.com), use this provider for IaC. Developers portal: [pathlyhq.com/en/developers](https://pathlyhq.com/en/developers).


## Related packages

| Package | Role |
|---|---|
| [pathly-opentofu](https://github.com/pathlyhq/pathly-opentofu) | OpenTofu docs & examples |
| [pathly-cdktf](https://github.com/pathlyhq/pathly-cdktf) | CDK for Terraform |
| [pathly-pulumi](https://github.com/pathlyhq/pathly-pulumi) | Pulumi |
| [pathly-ansible](https://github.com/pathlyhq/pathly-ansible) | Ansible |
| [pathly-sdk-go](https://github.com/pathlyhq/pathly-sdk-go) | Go SDK |
| [pathly-sdk-python](https://github.com/pathlyhq/pathly-sdk-python) | Python SDK |
| [Pathly product](https://pathlyhq.com) | [Pathly monitoring](https://pathlyhq.com) |
## About Pathly

[Pathly](https://pathlyhq.com) is synthetic monitoring for agencies and e-commerce: replay the customer journey, catch broken checkouts before your clients call, and keep evidence (screenshot, step, runbook) ready for the invoice. Product: [pathlyhq.com](https://pathlyhq.com) · Developers: [pathlyhq.com/en/developers](https://pathlyhq.com/en/developers) · Status & pricing: [pathlyhq.com/en/pricing](https://pathlyhq.com/en/pricing).

## Author

| | |
|---|---|
| **Company** | Pathly |
| **Author** | Simon Raynaud / keyral |

See [AUTHORS](AUTHORS).
