# Changelog

## 0.1.7 — 2026-09-26

- Same Chain surface as 0.1.6, with hop coverage at 100% and FR/ES SEO pages so the release pipeline can publish.

## 0.1.6 — 2026-09-26

- `pathly_scenario.http_chain`: Chain hops (`httpChain` on `/v1/scenarios`), including hop headers (sensitive). Tag exists, registry publish did not.

## 0.1.0 — 2026-09-23

- First publish: Terraform / OpenTofu provider `pathlyhq/pathly`.
- Resources: `pathly_scenario`, `pathly_webhook`, `pathly_maintenance_window`, `pathly_sla_target`.
- Auth via `PATHLY_API_TOKEN` (optional `PATHLY_API_URL`).
- Idempotent creates and retries honouring `Retry-After`.
