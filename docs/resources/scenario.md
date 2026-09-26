---
page_title: "pathly_scenario"
description: |-
  An HTTP check or a browser journey.
---

# pathly_scenario

A Ping (HTTP check), a Flow (browser journey), or a Chain (HTTP hops).

A browser journey is write-only: the API never returns the steps, only
`scenario_fingerprint`. A `fill` or `http_auth` step can carry a password —
keep those values in a secret store, not in the repository. The state still
holds them, marked sensitive.

## Simple example — HTTP check

```terraform
resource "pathly_scenario" "home" {
  name         = "Home page"
  url          = "https://shop.example.com/"
  interval_sec = 300
  expect_text  = "Our products"
  severity     = "major"
  folder       = "Shop"
  tags         = ["prod"]
}
```

## Chain — login then GET /me

```terraform
resource "pathly_scenario" "api_me" {
  name         = "Login then /me"
  interval_sec = 300

  http_chain = [
    {
      name          = "Login"
      method        = "POST"
      url           = "https://api.example.com/login"
      body          = jsonencode({ email = var.api_user, password = var.api_password })
      assert_status = 200
      extract_json_path = "token"
      extract_json_as   = "token"
    },
    {
      name          = "Me"
      method        = "GET"
      url           = "https://api.example.com/me"
      assert_status = 200
    },
  ]
}
```

## Complex example — five-step checkout

The worker opens the login page, types the credentials, submits, and fails if
the cart heading is missing. The password comes from a variable so it does not
sit in the repository.

```terraform
variable "shop_user" {
  type = string
}

variable "shop_password" {
  type      = string
  sensitive = true
}

resource "pathly_scenario" "checkout" {
  name         = "Checkout funnel"
  type         = "browser"
  interval_sec = 60
  severity     = "critical"
  folder       = "Shop"
  tags         = ["prod", "payment"]
  runbook      = "https://wiki.example.com/ops/checkout-unavailable"

  viewport          = "desktop"
  locale            = "fr-FR"
  scenario_timezone = "Europe/Paris"
  click_delay_ms    = 500

  steps = [
    {
      op  = "goto"
      url = "https://shop.example.com/login"
    },
    {
      op       = "fill"
      selector = "input[name=email]"
      value    = var.shop_user
    },
    {
      op       = "fill"
      selector = "input[name=password]"
      value    = var.shop_password
    },
    {
      op       = "click"
      selector = "button[type=submit]"
      text     = "Sign in"
    },
    {
      op   = "assert_text"
      text = "Your cart"
    },
  ]
}

output "checkout_fingerprint" {
  description = "Changes if the journey is edited in the console."
  value       = pathly_scenario.checkout.scenario_fingerprint
}
```

The first step must be `goto`: the worker has to open a page before it can
click or fill anything.

## Schema

### Required

- `name` (String) — Name displayed in the console and in alerts. 1 to 120
  characters.
- `interval_sec` (Number) — Period between two runs, in seconds. `0` for a
  scenario driven by `cron` only. From 0 to 2,592,000.

### Optional

- `url` (String) — Monitored address, over http or https. Required on `http`.
  On `browser`, taken from the first `goto`.
- `type` (String) — `http` or `browser`. **Forces replacement**: changing the
  type would destroy the scenario history.
- `enabled` (Boolean) — When false, the scenario exists but does not run.
- `method` (String) — `GET` or `HEAD`. A monitor that posts or deletes would
  act on the site at every run. HTTP only.
- `expected_status` (Number) — Expected HTTP status, from 100 to 599.
- `max_latency_ms` (Number) — Above this, the run is a performance failure.
  From 100 to 600,000.
- `expect_text` (String) — Text expected in the HTTP response. 500 characters
  at most.
- `regions` (List of String) — Probe regions, 8 at most. When empty, Pathly
  picks the default region of the plan.
- `tags` (List of String) — Free-form tags, 10 at most.
- `folder` (String) — Folder used to organize the console, 60 characters at
  most.
- `severity` (String) — `critical`, `major` or `minor`. Drives escalation.
- `runbook` (String) — On-call instructions, attached to the alert. 2,000
  characters at most.
- `cron` (String) — Cron schedule, in addition to or instead of
  `interval_sec`.
- `http_chain` (List of Object) — Chain hops, 1 to 10. A Ping needs `url`.
  A Chain needs `http_chain` (the first hop supplies `url` if omitted).
  `body` and `headers` are sensitive and never returned by the API.
- `steps` (List of Object) — Browser journey, 1 to 50 actions, first one
  `goto`. Never returned by the API.
- `headers` (Map of String) — Extra headers of the first request, 10 at most.
  `Host`, `Content-Length`, `Connection` and `Transfer-Encoding` are refused.
- `click_delay_ms` (Number) — Pause between two consecutive clicks, 0 to
  30,000. 2000 when omitted, 0 for none.
- `viewport` (String) — `desktop`, `tablet`, `mobile`, `iphone_se`,
  `iphone_14`, `pixel_7`, `galaxy_s21` or `ipad_mini`.
- `locale` (String) — Browser locale, for example `fr-FR`.
- `scenario_timezone` (String) — IANA timezone of the browser, distinct from
  the organization timezone.
- `basic_auth` (Object) — HTTP authentication of the first request.
  `username` and `password` are sensitive.

### Nested schema for `http_chain`

| Field | Required | What it does |
|---|---|---|
| `method` | yes | `GET`, `POST`, `PUT`, `PATCH`, `HEAD` or `DELETE`. |
| `url` | yes | Hop address. |
| `name` | no | Label shown in the timeline. |
| `headers` | no | Hop headers. Sensitive. Never returned. |
| `body` | no | JSON body. Sensitive. Never returned. |
| `wait_ms` | no | Pause after the response, 0 to 30,000. |
| `assert_status` | no | Expected HTTP status. |
| `expect_text` | no | Substring expected in the response. |
| `extract_json_path` | no | JSON path to extract for the next hop. |
| `extract_json_as` | no | Variable name for the extracted value. |
| `extract_cookie` | no | Cookie name to keep for the next hop. |

### Nested schema for `steps`

Every field except `op` is optional at the schema level. Which ones are
required depends on the operation, and the provider names both the index and
the missing field when one is absent.

| `op` | Required fields | What it does |
|---|---|---|
| `goto` | `url` | Open this address. Must be first. |
| `click` | `selector` | Click the element. Optional `new_tab`, `text`, `href`, `retry_times`. |
| `hover` | `selector` | Hover the element. |
| `fill` | `selector`, `value` | Type into a field. `value` is sensitive. |
| `select` | `selector`, `value` | Choose an option. |
| `upload` | `selector`, `file` | Upload a named fixture. |
| `scroll` | `selector` | Scroll the element into view. |
| `switch_tab` | — | Switch to a tab. Optional `url_includes`. |
| `wait` | `ms` | Pause, 0 to 30,000 milliseconds. |
| `wait_for` | `selector` | Wait until the element exists. Optional `timeout_ms`. |
| `assert_text` | `text` | Fail if the text is missing. Optional `selector`, `ignore_case`, `regex`. |
| `assert_visible` | `selector` | Fail if the element is hidden. |
| `assert_no_cmp` | — | Fail if a cookie banner is still up. |
| `assert_url` | — | Check the address. Optional `includes`, `path`, `url_regex`, `ignore_hash`, `ignore_query`. |
| `press` | `key` | `Enter`, `Escape`, `Tab`, `ArrowDown`, `ArrowUp` or `Space`. |
| `wait_for_response` | `url_includes` | Wait for a matching request. Optional `status`, `timeout_ms`. |
| `wait_for_networkidle` | — | Wait until the network is quiet. |
| `if_visible` | `selector` | Continue only if the element is visible. |
| `if_text` | `text` | Continue only if the text is present. |
| `assert_amount` | — | Check a displayed amount. Optional `selector`, `currency`, `min`, `max`, `equals`. |
| `assert_json_path` | `path`, `json_equals` | Check a JSON value. |
| `assert_header` | `name`, `includes` | Check a response header. |
| `http_auth` | `username`, `password` | HTTP authentication. Both sensitive. |

### Computed

- `id` (String) — Identifier assigned by Pathly.
- `last_status` (String) — Verdict of the last known run.
- `muted_until` (String) — Mute deadline, set from the console or the API.
  Read only here: a mute is a temporary operational gesture, not a desired
  state.
- `created_at` (String) — Creation date.
- `scenario_fingerprint` (String) — Hash of the stored journey. Changes when
  the tree is edited, including from the console.

## Attributes left empty

The optional attributes are also computed: when absent from the configuration,
they keep the value Pathly chose at creation time, or the one set from the
console. The plan therefore stays empty.

The trade-off: **removing a line does not revert to the default**. Going from
`severity = "critical"` to nothing leaves the scenario on `critical`. To go
back to the default, set the value you want explicitly.

The same rule applies to `steps`. After an import the list is empty, and the
next apply writes the configuration tree over whatever the console had.

## Import

Bring an existing scenario into state. Prefer an `import` block (Terraform /
OpenTofu ≥ 1.5) checked into the repository; the CLI form is equivalent.

```terraform
import {
  to = pathly_scenario.checkout
  id = "mon_01H8ZK…"
}
```

```sh
terraform import pathly_scenario.checkout mon_01H8ZK…
```

The identifier can be read from the scenario URL in the console. After the
import of a browser journey, state the `steps` again: the API never returns
them. Full working example: [`examples/import`](../../examples/import/).
