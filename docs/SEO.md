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
  id = "mon_…"
}

import {
  to = pathly_webhook.alerts
  id = "wh_…"
}

import {
  to = pathly_maintenance_window.window
  id = "mw_…"
}

import {
  to = pathly_sla_target.uptime
  id = "sla_…"
}

import {
  to = pathly_settings.org
  id = "settings"
}
```

```sh
terraform import pathly_scenario.checkout mon_…
terraform import pathly_webhook.alerts wh_…
terraform import pathly_maintenance_window.window mw_…
terraform import pathly_sla_target.uptime sla_…
terraform import pathly_settings.org settings
```

Working example: [`examples/import`](../examples/import/). Per-resource pages
under [`docs/resources/`](resources/). Root [README](../README.md) covers scopes
and webhook secret / state warnings.

## Badges (GitLab CI placeholders)

Development CI runs on **GitLab**. After the public project path is final,
replace `GROUP/PROJECT` below (same pattern for OpenTofu / CDKTF mirrors).

```markdown
[![CI](https://gitlab.com/GROUP/PROJECT/badges/main/pipeline.svg)](https://gitlab.com/GROUP/PROJECT/-/pipelines)
[![Terraform Registry](https://img.shields.io/badge/registry-pathlyhq%2Fpathly-623CE4?logo=terraform)](https://registry.terraform.io/providers/pathlyhq/pathly)
[![Website](https://img.shields.io/badge/Website-pathlyhq.com-111827)](https://pathlyhq.com)
[![API](https://img.shields.io/badge/API-developers-2563eb)](https://pathlyhq.com/en/developers)
```

Suggested project names once mirrored:

| Repo (local) | Placeholder GitLab path |
|---|---|
| `pathly-terraform-provider` | `pathlyhq/pathly-terraform-provider` |
| `pathly-opentofu` | `pathlyhq/pathly-opentofu` |
| `pathly-cdktf` | `pathlyhq/pathly-cdktf` |

GitHub is only the **read-only shop window** for the public Terraform Registry
(webhook on releases). Do not point the primary CI badge at GitHub Actions.

## Registry publish checklist

Use this before every public version. Detail lives in the root README
(“Publishing”).

### Shared (once)

- [ ] RSA 4096 GPG key (no elliptic curves); passphrase stored securely
- [ ] Public key registered on [registry.terraform.io](https://registry.terraform.io) *and/or* HCP private registry
- [ ] GitLab CI/CD variables **masked and protected**: `GPG_PRIVATE_KEY`, `GPG_PASSPHRASE`, `GPG_FINGERPRINT`

### Channel A — public Terraform Registry (`pathlyhq/pathly`)

- [ ] GitHub org `pathlyhq` + public repo `terraform-provider-pathly` (empty history before first mirror push)
- [ ] GitLab → GitHub push mirror (tags included; “protected branches only” **unchecked**)
- [ ] `GITHUB_TOKEN` fine-grained, `contents: write` on that repo only, masked + protected
- [ ] Provider declared on the Registry (webhook installed)
- [ ] Tag `vX.Y.Z` → pipeline `archives` → `publish-github` (draft) → **human publish** of the GitHub release
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
