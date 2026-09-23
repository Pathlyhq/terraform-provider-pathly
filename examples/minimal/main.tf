/*
 * The smallest useful setup: one scenario checked every five minutes.
 *
 * The API key comes from PATHLY_API_TOKEN, not from a Terraform variable: read
 * from a variable, it ends up in plaintext in the state file.
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

provider "pathly" {}

resource "pathly_scenario" "site" {
  name         = "Home page"
  url          = "https://shop.example.com"
  interval_sec = 300
  expect_text  = "Our products"
  tags         = ["prod"]
}

output "scenario_id" {
  value = pathly_scenario.site.id
}
