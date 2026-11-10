/*
 * One map entry per client site.
 *
 * An agency that maintains several client sites describes each of them once, in
 * var.clients. Every entry gets the same three things: a Ping of the home page,
 * a Flow that walks the checkout funnel up to the payment page, and an
 * availability objective. One webhook receives the alerts of all of them.
 *
 * Adding a client is adding an entry. Removing a client is removing it.
 */

terraform {
  required_version = ">= 1.6"
  required_providers {
    pathly = {
      source  = "pathlyhq/pathly"
      version = "~> 0.1"
    }
  }
}

# No arguments: the key comes from PATHLY_API_TOKEN, never from a file.
provider "pathly" {}

locals {
  # The same tags on the Ping and the Flow of a client. "client:<key>" filters
  # and groups them, in the console and in the pathly_scenarios data source.
  client_tags = { for key, client in var.clients : key => ["terraform", "client:${key}"] }
}

# Ping: does the home page answer, with the expected text, fast enough?
resource "pathly_scenario" "home" {
  for_each = var.clients

  name            = "${each.value.name} - home page"
  url             = each.value.home_url
  interval_sec    = var.ping_interval_sec
  expect_text     = each.value.home_text
  expected_status = 200
  max_latency_ms  = var.max_latency_ms
  severity        = each.value.severity
  folder          = each.value.name
  tags            = local.client_tags[each.key]
}

# Flow: can a visitor reach the payment page? A real browser opens the product
# page, adds the product to the cart and opens the checkout. It stops there: no
# step submits a payment, and the only values typed are the test login.
resource "pathly_scenario" "checkout" {
  for_each = var.clients

  name         = "${each.value.name} - checkout up to payment"
  type         = "browser"
  interval_sec = var.flow_interval_sec
  severity     = each.value.severity
  folder       = each.value.name
  tags         = concat(local.client_tags[each.key], ["checkout"])

  viewport = "desktop"

  # The login steps are kept only for the clients that set login_url. Their
  # values come from var.test_logins, which is sensitive.
  steps = concat(
    [
      for step in [
        {
          op  = "goto"
          url = each.value.login_url
        },
        {
          op       = "fill"
          selector = "input[name=email]"
          value    = try(var.test_logins[each.key].username, null)
        },
        {
          op       = "fill"
          selector = "input[name=password]"
          value    = try(var.test_logins[each.key].password, null)
        },
        {
          op       = "click"
          selector = "button[type=submit]"
        },
      ] : step if each.value.login_url != null
    ],
    [
      {
        op  = "goto"
        url = each.value.product_url
      },
      {
        op       = "click"
        selector = each.value.add_to_cart_selector
      },
      {
        op  = "goto"
        url = each.value.checkout_url
      },
      {
        op   = "assert_text"
        text = each.value.checkout_text
      },
    ],
  )

  lifecycle {
    precondition {
      condition     = each.value.login_url == null || contains(nonsensitive(keys(var.test_logins)), each.key)
      error_message = "Client \"${each.key}\" sets login_url: add its test account to var.test_logins, under the same key."
    }
  }
}

# Availability objective of the home page, one per client. The warning comes at
# 80 % of the error budget, while there is still room to act.
resource "pathly_sla_target" "home" {
  for_each = var.clients

  scenario_id          = pathly_scenario.home[each.key].id
  name                 = "${each.value.name} - ${each.value.objective_pct} %"
  objective_pct        = each.value.objective_pct
  window_days          = 30
  exclude_maintenance  = true
  warn_at_budget_ratio = 0.8
}

# One webhook for every client. The signing secret is only returned on
# creation, so it lives in the state: encrypt the backend.
resource "pathly_webhook" "alerts" {
  url    = var.webhook_url
  events = ["run.failed", "run.recovered"]
}

# Inventory read back from the API, to spot what was created by hand in the
# console and escapes the review.
data "pathly_scenarios" "all" {
  depends_on = [pathly_scenario.home, pathly_scenario.checkout]
}
