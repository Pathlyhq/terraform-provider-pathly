# Changelog

## [0.1.8](https://github.com/Pathlyhq/terraform-provider-pathly/compare/v0.1.7...v0.1.8) (2026-11-10)


### Features

* manage Spaces, status pages and the account ([#6](https://github.com/Pathlyhq/terraform-provider-pathly/issues/6)) ([5e5d5b3](https://github.com/Pathlyhq/terraform-provider-pathly/commit/5e5d5b32cdc9b4a79765a97f18b1212a32dd0d28))


### Bug Fixes

* attestation de provenance, modules Go vérifiés et vulnérabilités corrigées ([#3](https://github.com/Pathlyhq/terraform-provider-pathly/issues/3)) ([0a0f5f4](https://github.com/Pathlyhq/terraform-provider-pathly/commit/0a0f5f4c9125d05753a91cdd87f2fdc0f82a1e25))

## 0.1.7 — 2026-09-26

- Same Chain surface as 0.1.6, with hop coverage at 100% and FR/ES SEO pages so the release pipeline can publish.

## 0.1.6 — 2026-09-26

- `pathly_scenario.http_chain`: Chain hops (`httpChain` on `/v1/scenarios`), including hop headers (sensitive). Tag exists, registry publish did not.

## 0.1.0 — 2026-09-23

- First publish: Terraform / OpenTofu provider `pathlyhq/pathly`.
- Resources: `pathly_scenario`, `pathly_webhook`, `pathly_maintenance_window`, `pathly_sla_target`.
- Auth via `PATHLY_API_TOKEN` (optional `PATHLY_API_URL`).
- Idempotent creates and retries honouring `Retry-After`.
