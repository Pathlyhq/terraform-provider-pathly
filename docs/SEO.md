# Pathly Terraform Provider — SEO, discovery & publish checklist

Canonical product site: **[https://pathlyhq.com](https://pathlyhq.com)**  
Developers API: **[https://pathlyhq.com/en/developers](https://pathlyhq.com/en/developers)**  
Registry source: **`pathlyhq/pathly`** on the [Terraform Registry](https://registry.terraform.io/providers/pathlyhq/pathly)

## Why this package exists

Official Terraform / OpenTofu provider for [Pathly](https://pathlyhq.com) synthetic
monitoring: manage HTTP scenarios, webhooks, maintenance windows and SLA targets
as code. Auth via `PATHLY_API_TOKEN` only (never bake the token into `.tf`).

## Import existing monitors

Bring console-created resources into state without rewrite. Prefer **`import`
blocks** (Terraform / OpenTofu ≥ 1.5); CLI is equivalent.

```hcl
import {
  to = pathly_scenario.checkout
  id = "3f2b8c1e-6d4a-4b7f-9c2e-1a5d8e7f0b34"
}

import {
  to = pathly_webhook.alerts
  id = "7a1c9e52-0b3d-4f68-8e21-5c4d9a6b2f10"
}

import {
  to = pathly_maintenance_window.window
  id = "c84e2d17-9a5b-4c36-b0f1-2e7d6a3c8b59"
}

import {
  to = pathly_sla_target.uptime
  id = "1d6f0a93-4e8c-4a25-9b7d-8f3c2e5a6d41"
}

import {
  to = pathly_settings.org
  id = "settings"
}
```

```sh
terraform import pathly_scenario.checkout 3f2b8c1e-6d4a-4b7f-9c2e-1a5d8e7f0b34
terraform import pathly_webhook.alerts 7a1c9e52-0b3d-4f68-8e21-5c4d9a6b2f10
terraform import pathly_maintenance_window.window c84e2d17-9a5b-4c36-b0f1-2e7d6a3c8b59
terraform import pathly_sla_target.uptime 1d6f0a93-4e8c-4a25-9b7d-8f3c2e5a6d41
terraform import pathly_settings.org settings
```

Identifiers are UUIDs. Read a scenario's identifier in the address of its page
in the console. For the other resources, read the `id` field returned by the
API or by the matching data source (`pathly_webhooks`,
`pathly_maintenance_windows`, `pathly_sla_targets`).

Working example: [`examples/import`](../examples/import/). Per-resource pages
under [`docs/resources/`](resources/). Root [README](../README.md) covers scopes
and webhook secret / state warnings.

## Badges

```markdown
[![CI](https://github.com/Pathlyhq/terraform-provider-pathly/actions/workflows/ci.yml/badge.svg)](https://github.com/Pathlyhq/terraform-provider-pathly/actions/workflows/ci.yml)
[![Terraform Registry](https://img.shields.io/badge/registry-pathlyhq%2Fpathly-623CE4?logo=terraform)](https://registry.terraform.io/providers/pathlyhq/pathly)
[![Website](https://img.shields.io/badge/Website-pathlyhq.com-111827)](https://pathlyhq.com)
[![API](https://img.shields.io/badge/API-developers-2563eb)](https://pathlyhq.com/en/developers)
```

## Registry publish checklist

Use this before every public version. Detail lives in the root README
(“Publishing”).

### Shared (once)

- [ ] RSA 4096 GPG key (no elliptic curves); passphrase stored securely
- [ ] Public key registered on [registry.terraform.io](https://registry.terraform.io) *and/or* HCP private registry
- [ ] Secrets of the `release` GitHub environment (main branch only): `GPG_PRIVATE_KEY`, `GPG_PASSPHRASE`

### Channel A — public Terraform Registry (`pathlyhq/pathly`)

- [ ] Public repo `Pathlyhq/terraform-provider-pathly`
- [ ] Provider declared on the Registry (webhook installed)
- [ ] Merge the release-please PR → draft release → job `goreleaser` attaches the signed archives and publishes the release
- [ ] Verify: `curl -s https://registry.terraform.io/v1/providers/pathlyhq/pathly/versions | jq '.versions[].version'`

### Channel B — HCP Terraform private registry (optional)

- [ ] `TFE_TOKEN` (team token with *Manage private registry*), masked + protected
- [ ] Public key registered; `TFE_GPG_KEY_ID` set
- [ ] Tag → `archives` → `publish-hcp`
- [ ] Consumers use `source = "app.terraform.io/Pathly/pathly"` + `terraform login` / `TF_TOKEN_app_terraform_io`

### Docs / SEO hygiene each release

- [ ] `docs/` regenerated or reviewed (`tfplugindocs`); English canonical
- [ ] Import sections present on every managed resource (CLI + `import` block)
- [ ] README badges / links still point at **pathlyhq.com** (never invent alternate domains)
- [ ] Changelog / tag message describes breaking vs additive changes
- [ ] No secrets in examples, CI logs, or commit history

## Get started

1. Sign up: [pathlyhq.com/en/login?mode=signup](https://pathlyhq.com/en/login?mode=signup)
2. Create an API key in the console (minimum scopes for the resources you manage)
3. `export PATHLY_API_TOKEN=sp_…`
4. Install `source = "pathlyhq/pathly"` and `terraform init` (or `tofu init`)

## Author

Pathly · Simon Raynaud / keyral
