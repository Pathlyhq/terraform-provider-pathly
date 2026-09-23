> **La versión inglesa es la de referencia.** Esta página traduce
> [`docs/data-sources/members.md`](../../../docs/data-sources/members.md), la única versión que lee el registro de Terraform. Si hay
> divergencia, prevalece la inglesa.
---
page_title: "pathly_members"
description: |-
  Members of the organization, read-only.
---

# pathly_members

Lists the members. There is no resource to invite or to remove one: a
Terraform file that grants ownership turns a repository write into an access
grant.

```terraform
data "pathly_members" "admins" {
  filter_role = "admin"
}

output "admin_emails" {
  value = [for m in data.pathly_members.admins.members : m.email]
}
```

`filter_role` is not validated against a closed list: a role added later on
the Pathly side stays usable without waiting for a release.

## Schema

### Optional

- `filter_role` (String) — Keeps only this role.

### Computed

- `members` (List of Object) — `id`, `email`, `name`, `role`.

Required scope: `members:read`.
