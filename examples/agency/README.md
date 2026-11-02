# Agency example

One Space per client, the client sites filed in it, and one status page per
client with the badge to paste on the client site.

```sh
export PATHLY_API_TOKEN="sp_…"
terraform init
terraform plan
terraform apply
```

Spaces must be open for the organization. Until then the API answers 404 on
every Space route, and the plan stops on that error.

## Scopes the key needs

```
org:read          org:write           (Spaces, account)
scenarios:read    scenarios:write
status:read       status:write        (status pages)
```

## Things to watch

- **Destroying a Space archives it.** Its scenarios leave it and keep running.
  Nothing is deleted.
- **Destroying a status page unpublishes it.** The API has no deletion: the page
  becomes private, keeps its address, and Terraform stops tracking it.
- **Publishing a page exposes the scenario names and their availability.** Keep
  `public_status = false` for a client who has not agreed to it.
- **The organization page** is also driven by `status_slug` and `status_public`
  on `pathly_settings`. Manage it in one place only, otherwise each apply undoes
  the other.
