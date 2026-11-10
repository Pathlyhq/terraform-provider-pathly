# Import example

Adopt monitors already created in the [Pathly](https://pathlyhq.com) console
without rewriting them. Uses Terraform / OpenTofu **`import` blocks** (≥ 1.5).

```sh
export PATHLY_API_TOKEN="sp_…"
# Set real ids (or -var / tfvars):
#   import_scenario_id, import_webhook_id, import_maintenance_window_id, import_sla_target_id
terraform plan    # or: tofu plan
```

Equivalent CLI (one resource at a time):

```sh
terraform import pathly_scenario.checkout 3f2b8c1e-6d4a-4b7f-9c2e-1a5d8e7f0b34
terraform import pathly_webhook.alerts 7a1c9e52-0b3d-4f68-8e21-5c4d9a6b2f10
terraform import pathly_maintenance_window.backup c84e2d17-9a5b-4c36-b0f1-2e7d6a3c8b59
terraform import pathly_sla_target.uptime 1d6f0a93-4e8c-4a25-9b7d-8f3c2e5a6d41
terraform import pathly_settings.org settings
```

Identifiers are UUIDs. Read a scenario's identifier in the address of its page
in the console. For the other resources, read the `id` field returned by the
API or by the matching data source (`pathly_webhooks`,
`pathly_maintenance_windows`, `pathly_sla_targets`).

## Notes

- After importing a **browser** scenario, restate `steps` in config (API never returns them).
- After importing a **webhook**, restate `url` and recover the HMAC secret from your vault.
- Settings id is always `settings` (singleton).
- Never commit `PATHLY_API_TOKEN` or real secrets.

Product: [pathlyhq.com](https://pathlyhq.com) · API: [pathlyhq.com/en/developers](https://pathlyhq.com/en/developers).
