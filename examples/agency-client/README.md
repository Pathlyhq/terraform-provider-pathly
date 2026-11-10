# Agency example: one map entry per client site

For an agency or a freelancer who maintains several client sites. Each client is
one entry in `var.clients`, and every entry gets the same monitoring:

- a **Ping** of the home page (status, expected text, latency);
- a **Flow** that opens a product page, adds the product to the cart and reaches
  the payment page in a real browser;
- an **availability objective** on the home page, with a warning at 80 % of the
  error budget;
- a folder named after the client, and the tag `client:<key>`, to filter and
  group the scenarios of one client.

One webhook, shared by every client, is called when a run fails and when it
recovers.

```sh
export PATHLY_API_TOKEN="sp_…"            # from your secret store, never a file
cp terraform.tfvars.example terraform.tfvars
terraform init
terraform plan
terraform apply
```

## Add, change or remove a client

**Add a client**: add an entry to `clients` in `terraform.tfvars`, open a pull
request, apply. The plan shows three resources to create and nothing else.

```hcl
clients = {
  # …existing entries…

  cycles = {
    name         = "Example Cycles"
    home_url     = "https://store.example.org/"
    product_url  = "https://store.example.org/bikes/city"
    checkout_url = "https://store.example.org/checkout"
  }
}
```

**Change a client**: edit its entry. A new URL, severity or objective is an
in-place update. Keep the key: it is part of the resource address, so renaming
it destroys the scenarios and creates new ones, without their run history.

**Remove a client**: remove its entry. The plan shows three resources to
destroy. The scenarios take their run history with them, so export what the
client should keep before you apply.

## The Flow pays nothing

The Flow stops when the payment page is displayed. No step clicks a pay button
or submits a form other than the login, and the only values typed are those of
a test account. Two rules keep it that way:

- create the test account for monitoring, on the client's site, with no saved
  payment method, and never reuse a real customer's login;
- if you adapt the steps, end on an assertion (`assert_text`, `assert_url`),
  not on a click.

The selectors (`add_to_cart_selector`, the login fields) and `checkout_text`
depend on each site. The defaults are placeholders: check them in the browser's
inspector, then set them per client.

## Test accounts are sensitive, so they have their own variable

A client that sets `login_url` needs a test account in `var.test_logins`, under
the same key. The plan refuses a `login_url` without one. The variable is
marked `sensitive` and kept apart from `clients`, because Terraform does not
let a sensitive map drive `for_each`.

Do not write it in `terraform.tfvars`. Supply it from your secret store:

```sh
export TF_VAR_test_logins='{"bookshop":{"username":"probe@example.com","password":"…"}}'
```

## The state holds secrets: encrypt the backend

- The **webhook signing secret** is only returned on creation, so Terraform has
  to keep it in the state.
- The **test passwords** typed by the Flow are in the state as well, marked
  sensitive.

Use a remote backend with encryption at rest and restricted access. This
example declares no backend, so the state is a local file until you add one:
do not commit it, and do not publish the secret as an `output`.

Verify the HMAC signature on the receiving side of the webhook. A delivery
names the scenario by its identifier and its name, not by its tags: the
`scenario_ids` output gives the identifiers of each client, and every scenario
name starts with the client name.

## Scopes the key needs

```
scenarios:read  scenarios:write
alerting:read   alerting:write      (webhook)
sla:read        sla:write
```

For a pipeline that only runs `terraform plan`, the `:read` scopes are enough.
Outbound webhooks are not part of every plan: on a plan without them, the API
refuses the creation of the webhook.

## Spot what was created by hand

The `scenarios_outside_terraform` output lists every scenario that exists in
Pathly but not in this code, with its identifier, name and folder. An empty
list means the code describes everything. A scenario that shows up there was
created in the console and escapes the review: import it, or delete it.

To adopt a site that was first set up by hand, add its entry to `clients`, then
import the existing scenario at the address of that entry before you apply. The
identifier can be read from the scenario URL in the console:

```sh
terraform import 'pathly_scenario.home["cycles"]' 00000000-0000-4000-8000-000000000000
```

After the import of a Flow, the next apply writes the steps of this code over
the ones recorded in the console: the API never returns them.

Run `terraform plan -detailed-exitcode` every night in CI. Exit code 2 means
the plan is not empty: something differs between this code and Pathly.

## Things to watch

- **Quotas are not checked here.** Every client adds two scenarios, one of them
  a browser journey. Pick `ping_interval_sec` and `flow_interval_sec` within
  what your plan allows.
- **Only `example.com`, `example.org` and `example.net` appear in this
  example.** They are reserved for documentation: replace them with the sites
  you are allowed to monitor.
- **`terraform destroy` deletes the monitoring** of every client at once.
