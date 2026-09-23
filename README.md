# Pathly Terraform Provider

**English** · [Français](README.fr.md) · [Español](README.es.md)

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
| `pathly_status_page` | Public status page |

Per-resource documentation in [`docs/`](docs/), complete examples in
[`examples/`](examples/). The same reference pages are translated in
[`translations/fr/`](translations/fr/) and [`translations/es/`](translations/es/).
English stays canonical: it is the only version the Terraform Registry renders,
so it is the one that prevails when a translation lags behind.

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

Development lives on GitLab, and the pipeline builds and signs the archives from
there. Where those archives go is a separate decision, and there are two
channels. They are independent: either works on its own, and the same signed
archives feed both.

| Channel | Source address | Who can install it | GitHub needed |
|---|---|---|---|
| Public Terraform Registry | `pathlyhq/pathly` | anyone | yes, a public mirror |
| HCP Terraform private registry | `app.terraform.io/Pathly/pathly` | members of the `Pathly` organization | no |

The public registry is the only one that gives discoverability, a rendered
documentation site and installation without credentials. It also comes with a
hard requirement: it authenticates through GitHub, its namespace **is** a GitHub
organization name, and it ingests versions through a webhook on the releases of a
public GitHub repository. There is no import from GitLab, no manual upload and no
API to push a version. GitHub therefore serves as a read-only shop window, fed by
a mirror, while the signing key never leaves GitLab.

### Shared setup

**1. Signing key.** Both registries reject elliptic curves, so the key has to be
RSA:

```sh
gpg --full-generate-key            # RSA type, 4096 bits, with a passphrase
gpg --list-secret-keys --keyid-format=long     # note the fingerprint
gpg --armor --export <fingerprint>             # public part, to paste into the registry
gpg --armor --export-secret-keys <fingerprint> | base64 -w0   # private part, for CI
```

**2. GitLab CI/CD variables**, all **masked and protected**. "Protected" is not
a convenience: without that attribute, the signing key is readable from any
branch, and therefore exfiltrable by a simple push.

| Variable | Contents |
|---|---|
| `GPG_PRIVATE_KEY` | Private part of the key, base64-encoded |
| `GPG_PASSPHRASE` | Passphrase of that key |
| `GPG_FINGERPRINT` | Fingerprint of the key |

### Channel A — public Terraform Registry

The GitHub repository name is not yours to choose: the source `pathlyhq/pathly`
requires a `pathlyhq` organization and a `terraform-provider-pathly` repository.

**1. GitHub repository.** Create the `pathlyhq` organization and the **public**
`terraform-provider-pathly` repository, empty, with no generated README and no
generated license. The mirror would refuse to push onto a diverged history.

**2. GitLab → GitHub mirror.** Under **Settings → Repository → Mirroring
repositories**, direction *Push*, URL
`https://github.com/pathlyhq/terraform-provider-pathly.git`, with a GitHub token
as the password. Leave "Mirror only protected branches" **unchecked**: without
the tags, the registry has nothing to read.

**3. CI/CD variable** `GITHUB_TOKEN`, masked and protected: a **fine-grained**
token with the `contents: write` permission on the `terraform-provider-pathly`
repository only, and a short expiry. A classic token would give access to the
whole organization.

**4. Declaration to the registry.** Sign in to
[registry.terraform.io](https://registry.terraform.io) with the GitHub account,
declare the public key under *User settings → Signing keys*, then *Publish →
Provider* and pick the repository. The webhook is installed at that moment.

### Channel B — HCP Terraform private registry

No GitHub in this path. A provider is published straight to the organization's
registry over its API — which is also the only way, since the HCP console and the
VCS connection only handle modules, never providers.

**1. CI/CD variable** `TFE_TOKEN`, masked and protected: an HCP Terraform token
belonging to a team that holds the *Manage private registry* permission. Prefer a
team token over a personal one, so the pipeline does not stop working the day its
author leaves.

**2. Register the public key** once, and keep the id it returns. For a private
registry the namespace is the organization name:

```sh
curl -sS -X POST "https://app.terraform.io/api/registry/private/v2/gpg-keys" \
  -H "Authorization: Bearer $TFE_TOKEN" \
  -H "Content-Type: application/vnd.api+json" \
  -d "$(jq -n --arg ns Pathly --arg key "$(gpg --armor --export <fingerprint>)" \
        '{data:{type:"gpg-keys",attributes:{namespace:$ns,"ascii-armor":$key}}}')" \
  | jq -r '.data.attributes["key-id"]'
```

**3. CI/CD variable** `TFE_GPG_KEY_ID` with that id. It is what ties a published
version to the key that signed it.

**4. Consumers** need a token for the host, since a private registry is
authenticated. `terraform login app.terraform.io` writes one, or set
`TF_TOKEN_app_terraform_io` in CI:

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

### For each version

```sh
git tag v0.1.0 && git push origin v0.1.0
```

The pipeline tests, validates the examples, then stops. Three manual jobs follow,
in this order: `archives` builds the eleven targets and signs the checksums,
after which `publish-github` and `publish-hcp` upload those same archives to
whichever channels you use. Neither publishing job needs the other.

`publish-github` creates the GitHub release as a **draft**, so the registry
ingests nothing until you publish it by hand. Two explicit gestures, and that is
deliberate: a published version is immediately consumed by customers'
`terraform init`, and can never be withdrawn from a registry.

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
signed.

On HCP, a version whose checksum files or platform binaries were not all uploaded
stays in place but unusable, and `terraform init` reports it as unavailable rather
than missing. Re-run `publish-hcp` on the tag: the calls are safe to repeat.
