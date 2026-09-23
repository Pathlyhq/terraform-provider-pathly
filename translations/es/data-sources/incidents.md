> **La versión inglesa es la de referencia.** Esta página traduce
> [`docs/data-sources/incidents.md`](../../../docs/data-sources/incidents.md), la única versión que lee el registro de Terraform. Si hay
> divergencia, prevalece la inglesa.
---
page_title: "pathly_incidents"
description: |-
  Open and resolved incidents of the organization.
---

# pathly_incidents

Lists incidents, every page walked. History, not desired state: a `for_each`
built on this listing would create and destroy resources as incidents open and
close.

```terraform
data "pathly_incidents" "open" {
  filter_status      = "open"
  filter_scenario_id = pathly_scenario.checkout.id
}

output "open_tickets" {
  value = [
    for i in data.pathly_incidents.open.incidents : {
      title = i.title
      ado   = i.work_item_url
    }
  ]
}
```

## Schema

### Optional

- `filter_status` (String) — `open` or `resolved`.
- `filter_scenario_id` (String) — Keeps only the incidents of this scenario.

### Computed

- `incidents` (List of Object) — `id`, `scenario_id`, `scenario_name`,
  `status`, `title`, `opened_at`, `resolved_at`, `postmortem`,
  `public_postmortem`, `opened_run_id`, `resolved_run_id`, `assignee_email`,
  `assignee_name`, `work_item_id`, `work_item_url`.

Required scope: `incidents:read`. A refusal surfaces as an error, not an empty
list.
