/*
 * Adopt console-created resources with Terraform / OpenTofu import blocks
 * (require ≥ 1.5). Replace the id values, then:
 *
 *   export PATHLY_API_TOKEN="sp_…"
 *   terraform plan   # or: tofu plan
 *
 * Auth stays in the environment — never in .tf or in this file.
 */

terraform {
  required_version = ">= 1.5"
  required_providers {
    pathly = {
      source  = "pathlyhq/pathly"
      version = "~> 0.1"
    }
  }
}

provider "pathly" {}

# --- Resources to adopt (must match what already exists in Pathly) ---

resource "pathly_scenario" "checkout" {
  name         = "Checkout"
  url          = "https://shop.example.com/cart"
  interval_sec = 300
  expect_text  = "Your cart"
  tags         = ["prod", "imported"]
}

resource "pathly_webhook" "alerts" {
  # URL is never returned by the API after create/import — restate it here.
  url    = var.webhook_url
  events = ["run.failed", "run.recovered"]
}

resource "pathly_maintenance_window" "backup" {
  weekday      = 7
  start_minute = 180
  duration_min = 120
  reason       = "Weekly backup"
}

resource "pathly_sla_target" "uptime" {
  scenario_id          = pathly_scenario.checkout.id
  name                 = "Checkout — 99.9 %"
  objective_pct        = 99.9
  window_days          = 30
  exclude_maintenance  = true
  warn_at_budget_ratio = 0.8
}

resource "pathly_settings" "org" {
  timezone    = "Europe/Paris"
  alert_email = var.alert_email
}

# --- Import blocks (Terraform 1.5+ / OpenTofu) ---
# Replace placeholder ids with values from the Pathly console URLs.

import {
  to = pathly_scenario.checkout
  id = var.import_scenario_id
}

import {
  to = pathly_webhook.alerts
  id = var.import_webhook_id
}

import {
  to = pathly_maintenance_window.backup
  id = var.import_maintenance_window_id
}

import {
  to = pathly_sla_target.uptime
  id = var.import_sla_target_id
}

import {
  to = pathly_settings.org
  id = "settings"
}
