/*
 * An agency estate: one Space per client, its sites filed in it, and a public
 * status page per client with the badge to paste on the client site.
 *
 * Needs Spaces to be open for the organization: until then the API answers 404
 * on every Space route, and the plan says so.
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

# Who the key belongs to, and how much room the plan leaves.
data "pathly_account" "this" {}

resource "pathly_space" "client" {
  for_each = var.clients

  name = each.value.name
  kind = "client"

  lifecycle {
    # Fails the plan before the Space that would exceed the plan is requested.
    # A null limit means the plan sets none, and a null quota that the key is
    # limited to a Space: both pass.
    precondition {
      condition = coalesce(
        try(data.pathly_account.this.quotas.spaces.limit, null),
        length(var.clients),
      ) >= length(var.clients)
      error_message = "The plan does not allow that many Spaces. See ${data.pathly_account.this.billing_url}."
    }
  }
}

resource "pathly_scenario" "site" {
  for_each = var.clients

  name         = "${each.value.name} — home page"
  url          = each.value.url
  interval_sec = 300
  expect_text  = each.value.expect_text
  severity     = "major"
  tags         = ["terraform", "client"]
  space_id     = pathly_space.client[each.key].id
}

# One page per client. Destroying it unpublishes it: the API has no deletion.
resource "pathly_status_page" "client" {
  for_each = var.clients

  space_id = pathly_space.client[each.key].id
  public   = each.value.public_status
}

output "badges" {
  description = "Badge and status page address of each client, to paste on their site."
  value = {
    for key, page in pathly_status_page.client : key => {
      badge  = page.badge_url
      status = page.url
    }
  }
}

output "flow_remaining" {
  description = "Flow the plan still allows, null when it sets no quota."
  value       = data.pathly_account.this.quotas.flow.remaining
}
