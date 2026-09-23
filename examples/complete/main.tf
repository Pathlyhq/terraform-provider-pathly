/*
 * A full estate: scenarios described as data, a weekly maintenance window,
 * availability objectives and an alerting webhook.
 *
 * What this example is really about, beyond syntax: describing the scenarios in
 * a variable rather than one block per resource. That is what lets you add one
 * without rereading the code, and keeps the same threshold and severity rules
 * for all of them.
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

# No arguments: the key comes from PATHLY_API_TOKEN, the URL from PATHLY_API_URL.
provider "pathly" {}

resource "pathly_scenario" "journey" {
  for_each = var.journeys

  name         = each.value.name
  url          = each.value.url
  interval_sec = each.value.interval_sec
  expect_text  = each.value.expect_text

  # One latency threshold per journey: checkout tolerates less than a cached
  # home page.
  max_latency_ms  = each.value.max_latency_ms
  expected_status = 200
  severity        = each.value.severity
  folder          = var.folder
  tags            = concat(["terraform", var.environment], each.value.tags)
  runbook         = each.value.runbook
}

# One availability objective per critical journey, alerting at 80 % of the error
# budget spent: past that, too little margin is left for the rest of the month.
resource "pathly_sla_target" "objective" {
  for_each = { for key, j in var.journeys : key => j if j.severity == "critical" }

  scenario_id          = pathly_scenario.journey[each.key].id
  name                 = "${each.value.name} — ${var.objective_pct} %"
  objective_pct        = var.objective_pct
  window_days          = 30
  exclude_maintenance  = true
  warn_at_budget_ratio = 0.8
}

# Weekly backup: failures inside this window do not spend the error budget,
# since the objectives exclude maintenance.
resource "pathly_maintenance_window" "backup" {
  weekday      = 7 # Sunday
  start_minute = 3 * 60
  duration_min = 120
  reason       = "Weekly backup"
}

# Announced migration: a one-off window, with an end. A window that drags on
# would hide real outages, and the API refuses anything beyond 90 days.
resource "pathly_maintenance_window" "migration" {
  count = var.migration_starts_at == null ? 0 : 1

  scenario_id = pathly_scenario.journey["checkout"].id
  starts_at   = var.migration_starts_at
  ends_at     = var.migration_ends_at
  reason      = "Checkout funnel migration"
}

# Alerting webhook. The signing secret is only returned on creation, so it lives
# in the state, which must be encrypted on the backend side.
resource "pathly_webhook" "alerts" {
  url    = var.webhook_url
  events = ["run.failed", "run.recovered"]
}

# Five-step browser journey. The password stays in a variable so it does not
# sit in the repository. After apply, only scenario_fingerprint is readable
# from the API: a console edit of the tree shows up as drift.
resource "pathly_scenario" "checkout_browser" {
  name         = "Checkout as a customer"
  type         = "browser"
  interval_sec = 300
  severity     = "critical"
  folder       = var.folder
  tags         = ["terraform", var.environment, "payment"]
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

resource "pathly_settings" "org" {
  timezone    = "Europe/Paris"
  alert_email = var.alert_email
}

# Inventory read back from the API: useful to check that no scenario created by
# hand in the console is hiding outside the code.
data "pathly_scenarios" "prod" {
  filter_tag = var.environment

  depends_on = [pathly_scenario.journey]
}

output "scenarios_outside_terraform" {
  description = "Scenarios carrying the environment tag but absent from the code."
  value = [
    for s in data.pathly_scenarios.prod.scenarios : s.name
    if !contains([for j in pathly_scenario.journey : j.id], s.id)
  ]
}

output "webhook_fingerprint" {
  description = "Fingerprint of the destination: changes if someone edits the URL outside Terraform."
  value       = pathly_webhook.alerts.url_fingerprint
}
