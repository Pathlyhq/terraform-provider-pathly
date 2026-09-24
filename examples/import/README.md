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
terraform import pathly_scenario.checkout mon_…
terraform import pathly_webhook.alerts wh_…
terraform import pathly_maintenance_window.backup mw_…
terraform import pathly_sla_target.uptime sla_…
terraform import pathly_settings.org settings
```

## Notes

- After importing a **browser** scenario, restate `steps` in config (API never returns them).
- After importing a **webhook**, restate `url` and recover the HMAC secret from your vault.
- Settings id is always `settings` (singleton).
- Never commit `PATHLY_API_TOKEN` or real secrets.

Product: [pathlyhq.com](https://pathlyhq.com) · API: [pathlyhq.com/en/developers](https://pathlyhq.com/en/developers).
